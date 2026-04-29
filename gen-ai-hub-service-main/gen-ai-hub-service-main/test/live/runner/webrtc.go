/*
 Copyright (c) 2025 Pegasystems Inc.
 All rights reserved.
*/

package runner

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
	"gopkg.in/hraban/opus.v2"
)

const (
	webrtcTimeout         = 30 * time.Second
	webrtcFastTimeout     = 2 * time.Second
	webrtcResponseTimeout = 120 * time.Second
	sampleRate            = 24000
	bytesPerSample        = 2
	rtcFrameDuration      = 20 * time.Millisecond
	rtcSamplesPerFrame    = sampleRate * 20 / 1000
	rtcBytesPerFrame      = rtcSamplesPerFrame * bytesPerSample
)

// RealtimeModels is the fallback list of realtime models to test
// when /models discovery is not available.
var RealtimeModels = []string{
	"gpt-realtime",
	"gpt-realtime-mini",
	"gpt-realtime-1.5",
}

// realtimeClientSecretsPath returns the client_secrets endpoint path for the given model.
func realtimeClientSecretsPath(model string) string {
	return "/openai/deployments/" + model + "/v1/realtime/client_secrets"
}

// realtimeCallsPath returns the realtime calls endpoint path for the given model.
func realtimeCallsPath(model string) string {
	return "/openai/deployments/" + model + "/v1/realtime/calls"
}

type realtimeEvent struct {
	Type       string `json:"type"`
	Transcript string `json:"transcript,omitempty"`
}

// webrtcSession holds the shared state for a WebRTC realtime session,
// used by both text and audio test paths.
type webrtcSession struct {
	pc              *webrtc.PeerConnection
	audioTrack      *webrtc.TrackLocalStaticSample
	dc              *webrtc.DataChannel
	opusEncoder     *opus.Encoder
	stopSilence     chan struct{}
	testDone        chan struct{}
	sessionCreated  chan struct{}
	sessionUpdated  chan struct{}
	responseDone    chan struct{}
	audioDone       chan struct{}
	eventsMu        sync.Mutex
	allEvents       []json.RawMessage
	transcript      string
	requestAudio    []byte // raw PCM sent on the audio track (24kHz/16-bit/mono)
	responseAudioMu sync.Mutex
	responseAudio   []*rtp.Packet // RTP packets received from the remote audio track
}

// setupWebRTCSession performs the shared WebRTC session setup (steps 1-6):
// 1. Fetches ephemeral token
// 2. Creates peer connection with audio track
// 3. Creates data channel and wires event handlers
// 4. Exchanges SDP offer/answer
// 5. Starts silence sender, waits for ICE connection and data channel
// 6. Waits for session.created
func setupWebRTCSession(t *testing.T, svcBaseURL, jwtToken, model, instructions string) *webrtcSession {
	t.Helper()

	s := &webrtcSession{
		testDone:       make(chan struct{}),
		sessionCreated: make(chan struct{}),
		sessionUpdated: make(chan struct{}),
		responseDone:   make(chan struct{}, 1),
		audioDone:      make(chan struct{}, 1),
	}

	// Step 1: Fetch ephemeral token
	logVerbosef("Fetching ephemeral token for model %s...\n", model)
	ephemeralToken := fetchClientSecret(t, svcBaseURL, jwtToken, model, instructions)
	logVerbosef("Ephemeral token received (%d chars)\n", len(ephemeralToken))

	// Step 2: Create WebRTC peer connection
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatalf("Failed to create peer connection: %v", err)
	}
	s.pc = pc

	// Create Opus encoder for PCM-to-Opus conversion on the audio track.
	// The WebRTC track uses Opus codec, so raw PCM must be encoded before sending.
	enc, err := opus.NewEncoder(sampleRate, 1, opus.AppVoIP)
	if err != nil {
		t.Fatalf("Failed to create Opus encoder: %v", err)
	}
	s.opusEncoder = enc

	// Add audio track (required by Azure OpenAI Realtime)
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio", "test-mic",
	)
	if err != nil {
		t.Fatalf("Failed to create audio track: %v", err)
	}
	if _, err = pc.AddTrack(audioTrack); err != nil {
		t.Fatalf("Failed to add track: %v", err)
	}
	s.audioTrack = audioTrack

	// Track remote audio — drain to prevent backpressure, and capture
	// RTP packets for saving as Ogg/Opus + raw RTP files.
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		for {
			pkt, _, err := track.ReadRTP()
			if err != nil {
				return
			}
			clone := pkt.Clone()
			s.responseAudioMu.Lock()
			s.responseAudio = append(s.responseAudio, clone)
			s.responseAudioMu.Unlock()
		}
	})

	// Create data channel
	dc, err := pc.CreateDataChannel("oai-events", nil)
	if err != nil {
		t.Fatalf("Failed to create data channel: %v", err)
	}
	s.dc = dc

	dcReady := make(chan struct{})
	dc.OnOpen(func() {
		close(dcReady)
	})

	// Wire up data channel message handler
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		select {
		case <-s.testDone:
			return
		default:
		}

		s.eventsMu.Lock()
		s.allEvents = append(s.allEvents, json.RawMessage(msg.Data))
		s.eventsMu.Unlock()

		var event realtimeEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			return
		}
		logVerbosef("  Event: %s\n", event.Type)

		switch event.Type {
		case "error":
			t.Logf("  Server error event: %s", string(msg.Data))
		case "session.created":
			select {
			case <-s.sessionCreated:
			default:
				close(s.sessionCreated)
			}
		case "session.updated":
			select {
			case <-s.sessionUpdated:
			default:
				close(s.sessionUpdated)
			}
		case "response.audio_transcript.done", "response.output_audio_transcript.done":
			s.eventsMu.Lock()
			s.transcript = event.Transcript
			s.eventsMu.Unlock()
			logVerbosef("  Transcript: %s\n", event.Transcript)
		case "response.audio.done", "output_audio.done", "output_audio_buffer.stopped":
			select {
			case s.audioDone <- struct{}{}:
			default:
			}
		case "response.done":
			// Only signal completion when the model produced a transcript.
			// The server-side VAD may emit empty auto-responses (e.g. on
			// silence-to-audio transitions) that must be ignored.
			s.eventsMu.Lock()
			hasTranscript := s.transcript != ""
			s.eventsMu.Unlock()
			if hasTranscript {
				select {
				case s.responseDone <- struct{}{}:
				default:
				}
			}
		}
	})

	iceConnected := make(chan struct{})
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		select {
		case <-s.testDone:
			return
		default:
		}
		logVerbosef("  ICE state: %s\n", state)
		if state == webrtc.ICEConnectionStateConnected {
			select {
			case <-iceConnected:
			default:
				close(iceConnected)
			}
		}
	})

	// Step 3: Create and send SDP offer
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatalf("Failed to create SDP offer: %v", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		t.Fatalf("Failed to set local description: %v", err)
	}
	<-webrtc.GatheringCompletePromise(pc)

	logVerbose("Sending SDP offer...")
	sdpAnswer := sendRealtimeConnect(t, svcBaseURL, jwtToken, ephemeralToken, pc.LocalDescription().SDP, model)

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  sdpAnswer,
	}); err != nil {
		t.Fatalf("Failed to set remote description: %v", err)
	}

	// Send Opus-encoded silence to keep track alive
	s.startSilence()

	// Step 4: Wait for ICE connection
	logVerbosef("Waiting for ICE connection (timeout %s)...\n", webrtcTimeout)
	select {
	case <-iceConnected:
		logVerbose("ICE connected")
	case <-time.After(webrtcTimeout):
		t.Fatalf("Timeout waiting for ICE connection after %s", webrtcTimeout)
	}

	// Step 5: Wait for data channel
	logVerbosef("Waiting for data channel (timeout %s)...\n", webrtcFastTimeout)
	select {
	case <-dcReady:
		logVerbose("Data channel ready")
	case <-time.After(webrtcFastTimeout):
		t.Fatalf("Timeout waiting for data channel after %s", webrtcFastTimeout)
	}

	// Step 6: Wait for session.created event
	logVerbosef("Waiting for session.created event (timeout %s)...\n", webrtcTimeout)
	select {
	case <-s.sessionCreated:
		logVerbose("session.created received - WebRTC realtime session established successfully")
	case <-time.After(webrtcTimeout):
		t.Fatalf("Timeout waiting for session.created event after %s", webrtcTimeout)
	}

	return s
}

// waitForResponse waits for the model's response.done event and asserts a
// non-empty transcript, then saves session logs. This covers the common
// teardown steps (8-10) shared by text and audio test paths.
func (s *webrtcSession) waitForResponse(t *testing.T) {
	t.Helper()

	// Step 8: Wait for response.done
	logVerbosef("Waiting for model response (timeout %s)...\n", webrtcResponseTimeout)
	select {
	case <-s.responseDone:
		logVerbose("response.done received")
	case <-time.After(webrtcResponseTimeout):
		t.Fatalf("Timeout waiting for response.done after %s", webrtcResponseTimeout)
	}

	// Wait for the server to signal that it finished sending audio.
	// The response.audio.done event arrives on the data channel after the
	// last audio RTP packet is queued, but RTP packets may still be in
	// flight. Wait for the signal, then do a short drain.
	logVerbosef("Waiting for audio stream to complete (timeout %s)...\n", webrtcResponseTimeout)
	select {
	case <-s.audioDone:
		logVerbose("response.audio.done received")
	case <-time.After(webrtcResponseTimeout):
		logVerbose("Timed out waiting for response.audio.done, draining what we have")
	}

	// Short drain: wait for RTP packets to settle (500ms of silence).
	s.responseAudioMu.Lock()
	prevCount := len(s.responseAudio)
	s.responseAudioMu.Unlock()
	stableChecks := 0
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		s.responseAudioMu.Lock()
		count := len(s.responseAudio)
		s.responseAudioMu.Unlock()
		if count > 0 && count == prevCount {
			stableChecks++
			if stableChecks >= 5 {
				break
			}
		} else {
			stableChecks = 0
		}
		prevCount = count
	}

	// Step 9: Assert transcript is non-empty
	s.eventsMu.Lock()
	finalTranscript := s.transcript
	collectedEvents := make([]json.RawMessage, len(s.allEvents))
	copy(collectedEvents, s.allEvents)
	s.eventsMu.Unlock()

	if finalTranscript == "" {
		logVerbosef("Expected non-empty transcript from model response\n")
		t.Fail()
	} else {
		logVerbosef("AI response: %s\n", finalTranscript)
	}

	// Step 10: Save session log files
	s.saveSessionLog(t, collectedEvents, finalTranscript)
}

// startSilence launches a goroutine that sends Opus-encoded silence frames on
// the audio track to keep the WebRTC connection alive. Call stopSilenceLoop to
// stop it. Safe to call multiple times (replaces the previous silence loop).
func (s *webrtcSession) startSilence() {
	s.stopSilence = make(chan struct{})
	go s.silenceLoop(s.stopSilence)
}

// stopSilenceLoop stops the current silence sender goroutine.
func (s *webrtcSession) stopSilenceLoop() {
	select {
	case <-s.stopSilence:
	default:
		close(s.stopSilence)
	}
}

// silenceLoop sends Opus-encoded silence frames until the stop channel is closed.
func (s *webrtcSession) silenceLoop(stop chan struct{}) {
	silencePCM := make([]int16, rtcSamplesPerFrame)
	opusBuf := make([]byte, 4000)
	for {
		select {
		case <-stop:
			return
		default:
			n, err := s.opusEncoder.Encode(silencePCM, opusBuf)
			if err != nil {
				return
			}
			_ = s.audioTrack.WriteSample(media.Sample{
				Data:     opusBuf[:n],
				Duration: rtcFrameDuration,
			})
			time.Sleep(rtcFrameDuration)
		}
	}
}

// cleanup closes channels and the peer connection. Call via defer after setupWebRTCSession.
func (s *webrtcSession) cleanup() {
	close(s.testDone)
	s.stopSilenceLoop()
	s.pc.Close() //nolint:errcheck // cleanup path, error is not actionable
}

// RunWebRTCRealtimeTest tests the full WebRTC realtime flow through the gateway:
// 1. Fetches an ephemeral token via client_secrets endpoint
// 2. Establishes a WebRTC peer connection with audio track and data channel
// 3. Sends the SDP offer via the realtime/calls endpoint
// 4. Verifies ICE connection and data channel open
// 5. Verifies session.created event is received
// 6. Sends a text message and waits for the model response
func RunWebRTCRealtimeTest(t *testing.T, svcBaseURL, jwtToken, model, instructions, userPrompt string) {
	t.Helper()

	s := setupWebRTCSession(t, svcBaseURL, jwtToken, model, instructions)
	defer s.cleanup()

	// Step 7: Send a text message
	logVerbose("Sending text message...")
	sendRealtimeTextMessage(t, s.dc, userPrompt)

	s.waitForResponse(t)
}

// RunWebRTCRealtimeAudioTest tests the full WebRTC realtime flow using synthesized
// speech audio on the WebRTC audio track instead of a text data channel message.
// It mirrors RunWebRTCRealtimeTest but replaces the text-sending step with:
// 1. Synthesize speech from the prompt text using platform TTS
// 2. Stop the silence sender, send audio frames on the track, resume silence
// 3. Commit the audio buffer and explicitly request a response
//
// Server-side VAD is disabled (turn_detection=null) because pre-recorded audio
// contains natural speech pauses that would cause VAD to trigger mid-sentence,
// splitting the input and producing confused partial responses.
func RunWebRTCRealtimeAudioTest(t *testing.T, svcBaseURL, jwtToken, model, instructions, userPrompt string) {
	t.Helper()

	// Fail fast with actionable error if TTS dependency is missing.
	checkTTSDependency(t)

	s := setupWebRTCSession(t, svcBaseURL, jwtToken, model, instructions)
	defer s.cleanup()

	// Disable server-side VAD via session.update on the data channel.
	// Setting turn_detection=null in client_secrets causes HTTP 500 from the
	// gateway, so we update the session after it's established instead.
	// The turn_detection field is nested under audio.input in the realtime API.
	logVerbose("Disabling server-side VAD via session.update...")
	sessionUpdateMsg := `{"type":"session.update","session":{"type":"realtime","audio":{"input":{"turn_detection":null}}}}`
	if err := s.dc.SendText(sessionUpdateMsg); err != nil {
		t.Fatalf("Failed to send session.update to disable VAD: %v", err)
	}

	// Wait for the server to acknowledge the session update before sending
	// audio. Without this, audio may arrive before the model has processed
	// the updated instructions, causing it to ignore the prompt content.
	logVerbosef("Waiting for session.updated confirmation (timeout %s)...\n", webrtcFastTimeout)
	select {
	case <-s.sessionUpdated:
		logVerbose("session.updated received")
	case <-time.After(webrtcFastTimeout):
		t.Fatalf("Timed out waiting for session.updated after %s", webrtcFastTimeout)
	}

	// Step 7: Synthesize and send audio
	logVerbosef("Synthesizing speech for prompt: %q\n", userPrompt)
	pcmData := synthesizeSpeech(t, userPrompt)
	s.requestAudio = pcmData
	logVerbosef("Synthesized %d bytes of PCM audio (%.1fs at %dHz)\n",
		len(pcmData), float64(len(pcmData))/float64(sampleRate*bytesPerSample), sampleRate)

	// Stop silence, send real audio, resume silence
	close(s.stopSilence)
	logVerbose("Sending audio on WebRTC track...")
	sendAudioFrames(t, s.audioTrack, pcmData, s.opusEncoder)
	logVerbose("Audio sent, resuming silence on track")
	s.startSilence()

	// With VAD disabled, the model won't auto-respond to audio. Manually
	// commit the audio buffer and request a response via the data channel.
	logVerbose("Committing audio buffer and requesting response...")
	commitMsg := `{"type":"input_audio_buffer.commit"}`
	if err := s.dc.SendText(commitMsg); err != nil {
		t.Fatalf("Failed to send input_audio_buffer.commit: %v", err)
	}
	createMsg := `{"type":"response.create"}`
	if err := s.dc.SendText(createMsg); err != nil {
		t.Fatalf("Failed to send response.create: %v", err)
	}

	s.waitForResponse(t)
}

// checkTTSDependency verifies that the platform TTS tool is available and fails
// with an actionable error message if it's missing. Call this early in audio
// tests to fail fast with clear instructions instead of a cryptic exec error.
func checkTTSDependency(t *testing.T) {
	t.Helper()

	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("say"); err != nil {
			t.Fatal("Required dependency missing: 'say' command not found.\n" +
				"  The 'say' command is built into macOS and should be available at /usr/bin/say.\n" +
				"  If it's missing, your PATH may be misconfigured or you're running in a restricted sandbox.")
		}
	case "linux":
		if _, err := exec.LookPath("espeak-ng"); err != nil {
			t.Fatal("Required dependency missing: 'espeak-ng' not found.\n" +
				"  Install it with: sudo apt install espeak-ng\n" +
				"  espeak-ng is used to synthesize speech for the WebRTC audio test.")
		}
	default:
		t.Fatalf("Unsupported platform %q for WebRTC audio tests.\n"+
			"  Speech synthesis requires macOS (say) or Linux (espeak-ng).", runtime.GOOS)
	}
}

// synthesizeSpeech converts text to raw PCM audio matching the WebRTC track
// format (24kHz, 16-bit signed little-endian, mono). Auto-detects the TTS
// engine from the platform: "say" on macOS, "espeak-ng" on Linux. Fatals on
// unsupported platforms or synthesis errors.
func synthesizeSpeech(t *testing.T, text string) []byte {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "live-test-speech-*.wav")
	if err != nil {
		t.Fatalf("Failed to create temp file for TTS: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	var cmd *exec.Cmd
	var needsResample bool

	switch runtime.GOOS {
	case "darwin":
		logVerbose("Using say for speech synthesis (voice: Samantha)")
		cmd = exec.Command("say", "-v", "Samantha", "-o", tmpPath,
			"--file-format=WAVE",
			fmt.Sprintf("--data-format=LEI16@%d", sampleRate),
			text,
		)
	case "linux":
		logVerbose("Using espeak-ng for speech synthesis (lang: en)")
		cmd = exec.Command("espeak-ng", "-v", "en", "-w", tmpPath, text)
		needsResample = true
	default:
		t.Fatalf("Unsupported platform %q for speech synthesis: requires macOS (say) or Linux (espeak-ng)", runtime.GOOS)
	}

	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Fatalf("TTS command %q failed: %v\noutput: %s", cmd.Path, runErr, string(output))
	}

	wavData, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read WAV file: %v", err)
	}

	// Strip the 44-byte WAV header to get raw PCM
	const wavHeaderSize = 44
	if len(wavData) <= wavHeaderSize {
		t.Fatalf("WAV file too small (%d bytes), expected > %d", len(wavData), wavHeaderSize)
	}

	pcm := wavData[wavHeaderSize:]

	if needsResample {
		srcRate := readWAVSampleRate(wavData)
		if srcRate == 0 {
			srcRate = 22050 // espeak-ng default
		}
		if srcRate != sampleRate {
			logVerbosef("Resampling from %dHz to %dHz\n", srcRate, sampleRate)
			pcm = resamplePCM16(pcm, srcRate, sampleRate)
		}
	}

	return pcm
}

// readWAVSampleRate reads the sample rate from bytes 24-27 of a WAV header.
func readWAVSampleRate(wav []byte) int {
	if len(wav) < 28 {
		return 0
	}
	return int(binary.LittleEndian.Uint32(wav[24:28]))
}

// resamplePCM16 resamples 16-bit signed little-endian mono PCM from srcRate to dstRate
// using linear interpolation.
func resamplePCM16(pcm []byte, srcRate, dstRate int) []byte {
	srcSamples := len(pcm) / bytesPerSample
	if srcSamples == 0 {
		return pcm
	}

	// Decode source samples
	src := make([]int16, srcSamples)
	for i := range src {
		src[i] = int16(binary.LittleEndian.Uint16(pcm[i*bytesPerSample : (i+1)*bytesPerSample]))
	}

	// Linear interpolation resample
	ratio := float64(srcRate) / float64(dstRate)
	dstSamples := int(float64(srcSamples) / ratio)
	dst := make([]byte, dstSamples*bytesPerSample)

	for i := 0; i < dstSamples; i++ {
		srcPos := float64(i) * ratio
		idx := int(srcPos)
		frac := srcPos - float64(idx)

		var sample int16
		if idx+1 < srcSamples {
			sample = int16(float64(src[idx])*(1-frac) + float64(src[idx+1])*frac)
		} else {
			sample = src[idx]
		}
		binary.LittleEndian.PutUint16(dst[i*bytesPerSample:(i+1)*bytesPerSample], uint16(sample))
	}

	return dst
}

// sendAudioFrames sends PCM audio data on the WebRTC track in 20ms frames,
// paced at real-time to avoid overwhelming the receiver.
func sendAudioFrames(t *testing.T, track *webrtc.TrackLocalStaticSample, pcmData []byte, enc *opus.Encoder) {
	t.Helper()

	totalFrames := len(pcmData) / rtcBytesPerFrame
	duration := time.Duration(totalFrames) * rtcFrameDuration
	logVerbosef("Streaming %d frames (%s of audio, %d-byte frames every %dms)",
		totalFrames, duration.Round(time.Millisecond), rtcBytesPerFrame, rtcFrameDuration.Milliseconds())

	opusBuf := make([]byte, 4000) // max Opus frame size
	start := time.Now()
	for i := 0; i+rtcBytesPerFrame <= len(pcmData); i += rtcBytesPerFrame {
		frameNum := i/rtcBytesPerFrame + 1
		pcmFrame := pcmData[i : i+rtcBytesPerFrame]

		// Convert []byte PCM (little-endian int16) to []int16 for the Opus encoder.
		samples := make([]int16, rtcSamplesPerFrame)
		for s := 0; s < rtcSamplesPerFrame; s++ {
			samples[s] = int16(binary.LittleEndian.Uint16(pcmFrame[s*2 : s*2+2]))
		}

		n, err := enc.Encode(samples, opusBuf)
		if err != nil {
			t.Fatalf("Opus encode failed on frame %d: %v", frameNum, err)
		}

		if err := track.WriteSample(media.Sample{
			Data:     opusBuf[:n],
			Duration: rtcFrameDuration,
		}); err != nil {
			t.Logf("Warning: failed to write frame %d/%d: %v", frameNum, totalFrames, err)
		}
		logVerbosef("  Sent frame %d/%d (offset %d, opus %d bytes)\n", frameNum, totalFrames, i, n)
		time.Sleep(rtcFrameDuration)
	}
	elapsed := time.Since(start)
	logVerbosef("Streamed %d/%d frames in %s (realtime factor %.2fx)",
		totalFrames, totalFrames, elapsed.Round(time.Millisecond), float64(duration)/float64(elapsed))
}

// sendRealtimeTextMessage sends a text message over the data channel and triggers a response.
func sendRealtimeTextMessage(t *testing.T, dc *webrtc.DataChannel, text string) {
	t.Helper()

	itemCreate := map[string]any{
		"type": "conversation.item.create",
		"item": map[string]any{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{
					"type": "input_text",
					"text": text,
				},
			},
		},
	}
	data, _ := json.Marshal(itemCreate)
	if err := dc.SendText(string(data)); err != nil {
		t.Fatalf("Failed to send conversation.item.create: %v", err)
	}

	respCreate := map[string]any{
		"type": "response.create",
	}
	respData, _ := json.Marshal(respCreate)
	if err := dc.SendText(string(respData)); err != nil {
		t.Fatalf("Failed to send response.create: %v", err)
	}
}

// saveSessionLog saves the collected events, transcript, and audio files.
func (s *webrtcSession) saveSessionLog(t *testing.T, events []json.RawMessage, transcript string) {
	t.Helper()

	testID := uuid.Must(uuid.NewV7()).String()
	safeName := strings.ReplaceAll(t.Name(), "/", "_")
	filePrefix := fmt.Sprintf("/tmp/live-test-%s-%s", testID, safeName)
	logVerbosef("Session log files: %s.webrtc.*\n", filePrefix)

	// Save events JSON
	eventsJSON, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal events: %v", err)
		return
	}
	eventsFile := filePrefix + ".webrtc.json"
	if err := os.WriteFile(eventsFile, eventsJSON, 0o600); err != nil {
		t.Errorf("Failed to write events file: %v", err)
	} else {
		logVerbosef("Events saved to %s\n", eventsFile)
	}

	// Save transcript
	transcriptFile := filePrefix + ".webrtc.transcript"
	if err := os.WriteFile(transcriptFile, []byte(transcript), 0o600); err != nil {
		t.Errorf("Failed to write transcript file: %v", err)
	} else {
		logVerbosef("Transcript saved to %s\n", transcriptFile)
	}

	// Save request audio as WAV (raw PCM from TTS synthesis)
	if len(s.requestAudio) > 0 {
		requestFile := filePrefix + ".webrtc.request.wav"
		if err := writeWAVFile(requestFile, s.requestAudio, sampleRate, 1, 16); err != nil {
			t.Errorf("Failed to write request audio: %v", err)
		} else {
			logVerbosef("Request audio saved to %s (%d bytes PCM)\n", requestFile, len(s.requestAudio))
		}
	}

	// Save response audio as Ogg/Opus (playable by afplay, ffplay, VLC, etc.)
	s.responseAudioMu.Lock()
	responsePackets := make([]*rtp.Packet, len(s.responseAudio))
	copy(responsePackets, s.responseAudio)
	s.responseAudioMu.Unlock()

	if len(responsePackets) > 0 {
		// Ogg/Opus container
		oggFile := filePrefix + ".webrtc.response.ogg"
		if err := writeOggOpusFile(oggFile, responsePackets); err != nil {
			t.Errorf("Failed to write response ogg: %v", err)
		} else {
			logVerbosef("Response audio saved to %s (%d RTP packets)\n", oggFile, len(responsePackets))
		}

		// Best-effort Opus→WAV decode via ffmpeg (if available on PATH).
		wavFile := filePrefix + ".webrtc.response.wav"
		if err := convertOggToWAV(oggFile, wavFile); err != nil {
			logVerbosef("Ogg→WAV conversion skipped (non-fatal): %v\n", err)
		} else {
			logVerbosef("Response WAV saved to %s\n", wavFile)
		}

		// Raw RTP — useful for low-level debugging with Wireshark / custom decoders.
		rtpFile := filePrefix + ".webrtc.response.rtp"
		if rtpData, err := marshalRTPPackets(responsePackets); err != nil {
			t.Errorf("Failed to marshal RTP packets: %v", err)
		} else if err := os.WriteFile(rtpFile, rtpData, 0o600); err != nil {
			t.Errorf("Failed to write response rtp: %v", err)
		} else {
			logVerbosef("Response RTP saved to %s (%d bytes)\n", rtpFile, len(rtpData))
		}
	}
}

// writeWAVFile writes raw PCM data as a WAV file with the specified format.
func writeWAVFile(path string, pcm []byte, wavSampleRate, channels, bitsPerSample int) error {
	byteRate := wavSampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(pcm)

	var buf bytes.Buffer
	write := func(data any) error {
		return binary.Write(&buf, binary.LittleEndian, data)
	}
	// RIFF header
	buf.WriteString("RIFF")
	if err := write(uint32(36 + dataSize)); err != nil {
		return fmt.Errorf("write RIFF size: %w", err)
	}
	buf.WriteString("WAVE")
	// fmt sub-chunk
	buf.WriteString("fmt ")
	if err := write(uint32(16)); err != nil {
		return fmt.Errorf("write fmt size: %w", err)
	}
	if err := write(uint16(1)); err != nil { // PCM format
		return fmt.Errorf("write audio format: %w", err)
	}
	if err := write(uint16(channels)); err != nil { //nolint:gosec // channels is always 1
		return fmt.Errorf("write channels: %w", err)
	}
	if err := write(uint32(wavSampleRate)); err != nil {
		return fmt.Errorf("write sample rate: %w", err)
	}
	if err := write(uint32(byteRate)); err != nil {
		return fmt.Errorf("write byte rate: %w", err)
	}
	if err := write(uint16(blockAlign)); err != nil { //nolint:gosec // blockAlign is always 2
		return fmt.Errorf("write block align: %w", err)
	}
	if err := write(uint16(bitsPerSample)); err != nil { //nolint:gosec // bitsPerSample is always 16
		return fmt.Errorf("write bits per sample: %w", err)
	}
	// data sub-chunk
	buf.WriteString("data")
	if err := write(uint32(dataSize)); err != nil {
		return fmt.Errorf("write data size: %w", err)
	}
	buf.Write(pcm)

	return os.WriteFile(path, buf.Bytes(), 0o600)
}

// writeOggOpusFile writes RTP packets containing Opus audio into an Ogg/Opus
// container file, playable by ffplay, VLC, and most audio players.
// The Opus RTP clock rate is always 48000 Hz per RFC 7587, regardless of the
// internal SILK/CELT sample rate.
func writeOggOpusFile(path string, packets []*rtp.Packet) error {
	ogg, err := oggwriter.New(path, 48000, 1)
	if err != nil {
		return fmt.Errorf("create ogg writer: %w", err)
	}
	for _, pkt := range packets {
		if err := ogg.WriteRTP(pkt); err != nil {
			ogg.Close() //nolint:errcheck // best-effort cleanup
			return fmt.Errorf("write RTP packet: %w", err)
		}
	}
	return ogg.Close()
}

// marshalRTPPackets serializes RTP packets into a length-prefixed byte stream.
// Each packet is preceded by its length as a uint32 (little-endian), making the
// file reliably parseable since RTP packets are variable-length.
func marshalRTPPackets(packets []*rtp.Packet) ([]byte, error) {
	var buf bytes.Buffer
	for _, pkt := range packets {
		raw, err := pkt.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal RTP packet seq=%d: %w", pkt.SequenceNumber, err)
		}
		if err := binary.Write(&buf, binary.LittleEndian, uint32(len(raw))); err != nil {
			return nil, fmt.Errorf("write RTP packet length seq=%d: %w", pkt.SequenceNumber, err)
		}
		buf.Write(raw)
	}
	return buf.Bytes(), nil
}

// convertOggToWAV converts an Ogg/Opus file to a 24kHz mono 16-bit WAV file
// using ffmpeg. If ffmpeg is not installed, it returns an error and the caller
// should skip gracefully. This is best-effort — the .ogg file is always the
// primary response audio artifact.
//
// The silenceremove filter trims leading silence (below -40dB) so the WAV
// starts at the first audible sample. The model typically sends a few hundred
// milliseconds of silence before it begins speaking.
func convertOggToWAV(oggPath, wavPath string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found on PATH: install ffmpeg to enable Ogg→WAV conversion")
	}

	//nolint:gosec // ffmpegPath is from LookPath, oggPath/wavPath are test-generated temp paths.
	cmd := exec.Command(ffmpegPath, "-y", "-i", oggPath,
		"-af", "silenceremove=start_periods=1:start_silence=0.05:start_threshold=-40dB",
		"-ar", "24000", "-ac", "1", "-sample_fmt", "s16", "-f", "wav", wavPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w\noutput: %s", err, string(output))
	}
	return nil
}

// fetchClientSecret calls the client_secrets endpoint to get an ephemeral token.
func fetchClientSecret(t *testing.T, svcBaseURL, jwtToken, model, instructions string) string {
	t.Helper()

	sessionConfig := map[string]any{
		"session": map[string]any{
			"type":         "realtime",
			"model":        model,
			"instructions": instructions,
			"audio": map[string]any{
				"output": map[string]any{
					"voice": "echo",
				},
			},
		},
	}
	body, err := json.Marshal(sessionConfig)
	if err != nil {
		t.Fatalf("Failed to marshal session config: %v", err)
	}

	reqURL := svcBaseURL + realtimeClientSecretsPath(model)
	logVerbosef("POST %s\n", reqURL)
	logVerbosef("Request body: %s\n", string(body))

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	client := &http.Client{Timeout: webrtcTimeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Client secrets request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Client secrets returned HTTP %d: %s\nModel: %s, Voice: echo", resp.StatusCode, string(respBody), model)
	}

	// Response can be either:
	// - Direct Azure shape: {value: "ek_...", session: {...}}
	// - APIM/wrapped shape: {client_secret: {value: "ek_...", ...}}
	var parsed map[string]any
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		t.Fatalf("Failed to parse client secret response: %v", err)
	}

	// Try direct shape first
	if val, ok := parsed["value"].(string); ok && val != "" {
		return val
	}
	// Try wrapped shape
	if cs, ok := parsed["client_secret"].(map[string]any); ok {
		if val, ok := cs["value"].(string); ok && val != "" {
			return val
		}
	}

	t.Fatalf("No ephemeral token found in response: %s", string(respBody))
	return ""
}

// sendRealtimeConnect sends the SDP offer to establish a WebRTC session.
func sendRealtimeConnect(t *testing.T, svcBaseURL, jwtToken, ephemeralToken, sdpOffer, model string) string {
	t.Helper()

	reqURL := fmt.Sprintf("%s%s?model=%s", svcBaseURL, realtimeCallsPath(model), model)
	logVerbosef("POST %s\n", reqURL)

	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(sdpOffer))
	if err != nil {
		t.Fatalf("Failed to create connect request: %v", err)
	}
	req.Header.Set("Content-Type", "application/sdp")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("X-Ephemeral-Token", ephemeralToken)

	client := &http.Client{Timeout: webrtcTimeout}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Connect request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read connect response: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("Connect returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody)
}
