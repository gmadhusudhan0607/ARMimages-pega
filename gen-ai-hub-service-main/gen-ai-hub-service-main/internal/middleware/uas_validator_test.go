/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Pega-CloudEngineering/gen-ai-hub-service/internal/cntx"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/jwk"
)

func TestUASMiddleware(t *testing.T) {
	jwkJSON := `{
		"keys": [ 
			{"kty":"RSA","e":"AQAB","use":"sig","kid":"166d5aea80e9b53048ecac2f9d06a372","alg":"RS256","n":"lkXZfxl1iuLtT0DQ6hVRyxiAQwt1ZWHBmd4ZLEI8ttnJGC_-a8b1ofyqcTcI83n_jbBQVEyIwyLp7T5I8-Rt5AHznJD2KcyPJX8my82T_1O717dUUoWMpJ3IVF5SRb-o36ipPpBwrsmiGM6v0cXN80sx_86LnD7EK9tuTfNQgPFE8lYmCon3Fh3pAJa2RRMGeEAX_PJzwbmeSZDPRjdC_QvgPUeBBoRnXN4OOixkPGblGNNIIdkhtomap6vDZIWCeh1-5SqrzBybbZYAXckoSqYdsY2mJ8JY5daEqZXBxtmAyg7BB7cNNREIeHo2J3P2iVF9EfDjbte7ZiNtosG7anpz42CdngwyvoN4p-XczYzHzB9100FpB4h9iLwtoZjaGUn1py_lwy05g9bC-NngSxZSqtQxYE7xRLuecWK9dkNIaaIcvtPxOUNFgmfmVbwGqLMyNzui8hhOgVSXvEq2RzxjFTFDlL0hnFyRfxYpdaXNpwOjYJ4Fr8hbSWtJv3FpubEpKdgVpagQkgv-CrgdXsVGYL1wxsdPf2OUKWLlhiZn5XXFMtoqWhZkT-ic_SS6B2ePPir3TpUPXwPET-ijIlmpfvi5Fqs_oXPFxsDsHBcSeYxNorqk7mHAdqMTxBjQsOzDOG-jSAD84iuvze1jumnTTymQ0xmiRJaYdRDGeyk"}
		]
	  }
	  `
	expiredTokenError := `{"error":"exp not satisfied"}`
	set, err := jwk.Parse([]byte(jwkJSON))
	if err != nil {
		t.Error("Error parsing token")
	}
	gocache.Set("uas-jwks", set, -1)

	os.Setenv("PLATFORM_TYPE", "launchpad")
	ctx := cntx.ServiceContext("genai-hub-service")

	// create a new Gin router
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Add the AuthMiddleware to the router
	r.Use(UasValidator(ctx))

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "This is the root page")
	})

	// create a request with a empty Authorization header
	req1 := httptest.NewRequest("GET", "/", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	// check if the status is 401(Unauthorized) when there is no authorization token present
	if w1.Code != http.StatusUnauthorized {
		t.Errorf("Case1: Request with no Authorization token. Expected response status code: %v, received status code: %v\n", http.StatusUnauthorized, w1.Code)
	}

	// Create a request with not parsable jwt token
	authToken := "MyPassKey1234567"
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("Authorization", authToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	// Check if the response status code is 401 when not parsable jwt token is provided
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Case1: Request with non bearer token. Expected response status code: %v, received status code: %v", http.StatusUnauthorized, w2.Code)
	}

	// Create a request with a expired authorization token
	bearerToken := "Bearer eyJraWQiOiIxNjZkNWFlYTgwZTliNTMwNDhlY2FjMmY5ZDA2YTM3MiIsImFsZyI6IlJTMjU2In0.eyJ0ZW5hbnRfaWQiOiJhMjFhMTkwOS1jNjY0LTQxYmUtYmZjNS05YTNjZTBkYWU1MmMiLCJzdWIiOiJtYW5vamt1bWFydmFybWEucGVubWF0c2FAaW4ucGVnYS5jb20iLCJwZXJzb25hIjpbIlBST1ZfREVWIiwiUFJPVl9BRE1JTiJdLCJ1c2VyX25hbWUiOiIwMHVjYTRmbzB6T1dldDBpbzM1NyIsImlzcyI6InVybjp1YXMtc2VydmljZSIsImlzb2xhdGlvbi1pZCI6ImEyMWExOTA5LWM2NjQtNDFiZS1iZmM1LTlhM2NlMGRhZTUyYyIsImNsaWVudF9pZCI6IjF1M3EzbXpiTXBGanpZOFQiLCJhdXRob3JpdGllcyI6WyJPSURDX1VTRVIiLCJTQ09QRV9wZWdhdXNlciIsIlNDT1BFX3BlZ2Fyb2xlcyIsIlNDT1BFX29wZW5pZCIsIlNDT1BFX2VtYWlsIiwiU0NPUEVfcHJvZmlsZSJdLCJvcGVyYXRvciI6Im1hbm9qa3VtYXJ2YXJtYS5wZW5tYXRzYUBpbi5wZWdhLmNvbSIsImF1ZCI6InByb2QiLCJvcGVyYXRvcl9uYW1lIjoiTWFub2ogUGVubWF0c2EiLCJuYmYiOjE3MjAxNDgyMDAsInNjb3BlIjpbInVzZXJfaW5mbyJdLCJpc29sYXRpb24taWRzIjp7InJlYWQiOlsiYTIxYTE5MDktYzY2NC00MWJlLWJmYzUtOWEzY2UwZGFlNTJjIl0sIndyaXRlIjpbImEyMWExOTA5LWM2NjQtNDFiZS1iZmM1LTlhM2NlMGRhZTUyYyJdfSwiZXhwIjoxNzIwMTUxODMwLCJpYXQiOjE3MjAxNDgyMzAsImp0aSI6ImJlYWU0NGYxLWM2MDctNGZkYy05MzlhLWYxYTIzOWRlODZmZiJ9.CG8eOww6V9ZSll3Vtr5VMVoyE_3MnfTYwk4PZaarNVsR0w0M2jaDXGPGSwAmwPs7dOWlzaqn0emGM0i_x8c5MVntfZTNXRwDAmSqLyh6lUy-BxC-hQF5ZDfDPznCbwq9ao8q_DShoPJSNhIheXponbg1z5oeceeL_BUD1fBBWAAViX3ZvUwX7NEiUSVv4G27ODqw07rtYpnoY0vnCClhne7QThg9MIa6GBs6yaKsvSvvAXWrPREgO8wT6XH84WlpTaqHqwOjTufIbCNRQUq-HIW54a0zGRLd50Odxzhk2fJvYfGlFk2WN7LvUjCesXfcpofNlNPlQCPRD7oT0yvw42u4ub_vvPlG7dazk7I9K82R9cCp1ecsRIk_aTaRDGJIaDmDj4BEuNl-i-MujJH2SEIE_gOEOR8t_t14UiFXcT7MOlawkh5hQ-P9UktKhxAL8ZAjtxay84Okk8AXZX2oZKTD_lD65CoKWKCk_iS7rdhCu9s_8wDvD85wafwz7E3eXGrnpqEuWGMg8qmGH-PhtSBxitQa_whbXwplCqSwsFuAvn0Wezcsmu-cI3j9THcEy5RYVFA0kB0yw6l61_hu8ypThM2_tO4nntP9KMiCKEILeottX6v11Z1RTsoruKf2m1-5un-cJmIhfO_4TxwVB622Umvrsp5bYtrZrXranQI"
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.Header.Set("Authorization", bearerToken)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)

	// Check if the response status code is 401 when expired bearer token is provided
	body, _ := io.ReadAll(w3.Body)
	if string(body) != expiredTokenError {
		t.Errorf("Case1: Request with expired bearer token. Expected response status code: %v, received status code: %v", http.StatusUnauthorized, w3.Code)
	}

}
