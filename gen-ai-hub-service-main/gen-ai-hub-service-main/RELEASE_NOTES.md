# Changelog
---
<br>
All notable changes to this project will be documented in this file.
<br>
<br>

<Unreleased>

<a name='1.49.1-20260420103102'></a>
### [1.49.1-20260420103102 (2026-04-20)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.49.1-20260420103102)
### No Notable Updates Found

---
<br>



### Internal / Tooling

- Consolidated agent and instruction files onto a single source of truth: `.github/copilot-instructions.md` (with `AGENTS.md` as a relative symlink) and per-agent definitions under `.github/agents/`. Removed `.claude/` and `CLAUDE.md`; migrated the `jarvis` slash command to `.github/prompts/jarvis.prompt.md`. Added `.github/workflows/instructions-drift-check.yml` to prevent future drift.

<a name='1.49.0-20260414112235'></a>
### [1.49.0-20260414112235 (2026-04-14)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.49.0-20260414112235)
### New Functionality

- [US-740014](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-740014): Remove -next and -deprecated labels - [PR 455](https://github.com/pega-cloudengineering/gen-ai-hub-service/pull/455)
- [US-727521](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-727521): metric reasoning tokens - [PR 446](https://github.com/pega-cloudengineering/gen-ai-hub-service/pull/446)
- [US-740710](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-740710): govern api version - [PR 464](https://github.com/pega-cloudengineering/gen-ai-hub-service/pull/464)
- [US-724845](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-724845): Resolve empty migration log issue (no-op) - [PR 480](https://github.com/pega-cloudengineering/gen-ai-hub-service/pull/480)

### Completed Tasks

- [TASK-1876289](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/task/TASK-1876289): portable agent memory paths - [PR 456](https://github.com/pega-cloudengineering/gen-ai-hub-service/pull/456)

### Resolved Bugs

- [BUG-984011](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-984011): Gemini 3.1 Flash Image Preview model mapping using wrong target - [PR 463](https://github.com/pega-cloudengineering/gen-ai-hub-service/pull/463)

### Addressed Issues

- [ISSUE-141152](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-141152): docs: rewrite README.md with GenAI Hub Service-specific content - [PR 494](https://github.com/pega-cloudengineering/gen-ai-hub-service/pull/494)

---
<br>



<a name='1.48.1-20260316154056'></a>
# [1.48.1-20260316154056 (2026-03-16)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.1-20260316154056)
### Resolved Bugs

- [BUG-981893](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-981893): -fix-metrics-parsing-for-llama - [PR 438](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/438)

---
<br>



<a name='1.48.0-20260309143223'></a>
# [1.48.0-20260309143223 (2026-03-13)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260309143223)
### No Notable Updates Found

---
<br>



<a name='1.48.0-20260312124450'></a>
# [1.48.0-20260312124450 (2026-03-12)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260312124450)
### Resolved Bugs

- [BUG-979804](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-979804): -bump-up-go-and-sax-client - [PR 430](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/430)

---
<br>



<a name='1.48.0-20260309143223'></a>
# [1.48.0-20260309143223 (2026-03-09)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260309143223)
### New Functionality

- [US-735850](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-735850): -adjusting deprecation date for gpt-4o and 4o-mini - [PR 425](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/425)
- [US-728594](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-728594): Remove obsolete AWS Lambda and GCP Cloud Functions distribution files - [PR 420](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/420)
- [US-731839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731839): Improve model selection - [PR 415](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/415)

### Resolved Bugs

- [BUG-975203](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975203): Reuse HTTP proxy clients to prevent connection/memory leak - [PR 416](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/416)
- [BUG-975203](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975203): Update instructions - [PR 421](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/421)
- [BUG-975203](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975203): Add memory leak detection to live tests - [PR 418](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/418)

---
<br>



<a name='1.48.0-20260306090429'></a>
# [1.48.0-20260306090429 (2026-03-06)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260306090429)
### New Functionality

- [US-735850](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-735850): -update-gpt-4o-and-mini-deprec-date - [PR 417](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/417)

---
<br>



<a name='1.48.0-20260305085640'></a>
# [1.48.0-20260305085640 (2026-03-05)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260305085640)
### New Functionality

- [US-731839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731839): - Use Converse API for Amazon models - [PR 413](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/413)
- [US-731839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731839): Execute test for all embedding models - [PR 407](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/407)

### Resolved Bugs

- [BUG-979651](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-979651): Fix missing headers and Content-Length error during retry responses - [PR 412](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/412)
- [BUG-975203](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975203): - Add pprof profiling endpoints, fix memory leak, and optimize metadata loading - [PR 410](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/410)
- [BUG-979651](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-979651): Check HTTP headers during live tests - [PR 411](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/411)

---
<br>



<a name='1.48.0-20260228212404'></a>
# [1.48.0-20260228212404 (2026-03-01)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260228212404)
### No Notable Updates Found

---
<br>



<a name='1.48.0-20260227085624'></a>
# [1.48.0-20260227085624 (2026-02-27)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260227085624)
### New Functionality

- [US-732774](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-732774): Adds Opus 4.6 to Infra SCE options - [PR 402](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/402)
- [US-733645](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-733645): default values - [PR 405](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/405)
- [US-731839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731839): Improve streaming response validation robustness - [PR 406](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/406)
- [US-731839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731839): Run live tests againt environments in k8s - [PR 404](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/404)
- [US-731839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731839): Improve usability of live tests - [PR 403](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/403)
- [US-732311](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-732311): [GW] Onboard Anthropic Sonnet 4.6 - [PR 400](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/400)
- [US-730366](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-730366): Allow model InferenceRegion to be different than the GenAI Infra region - [PR 393](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/393)
- [US-731839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731839): Live tests - [PR 394](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/394)
- [US-729418](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-729418): -Add Pro model override support to GenAI Hub Service SCE configuration - [PR 397](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/397)

### Resolved Bugs

- [BUG-971917](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-971917): Disable buffering for streaming requests - [PR 399](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/399)

---
<br>



<a name='1.47.5-20260218114618'></a>
# [1.47.5-20260218114618 (2026-02-18)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.47.5-20260218114618)
### Resolved Bugs

- [BUG-977197](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-977197): Onboard Anthropic Sonnet 4.6 - [PR 395](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/395)

---
<br>



<a name='1.48.0-20260217130459'></a>
# [1.48.0-20260217130459 (2026-02-18)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.48.0-20260217130459)
### New Functionality

- [US-729416-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-729416-1): -add-pro-label - [PR 383](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/383)
- [US-721982-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-721982-1): - Show/Hide Preview Models - [PR 388](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/388)
- [US-716759-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-716759-1): - Integrate Gemini 3 Pro - [PR 382](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/382)
- [US-721983](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-721983): - Integrate Gemini 3 Flash - [PR 381](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/381)
- [US-684490-3](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-684490-3): Make efficent use of AWS credentials and SAX tokens - [PR 357](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/357)
- [US-724887](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-724887): Add Nova Embedding - Documentation - [PR 379](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/379)
- [US-724887](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-724887): - Add Nova Embedding - [PR 377](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/377)
- [US-727607](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-727607): -  Add AWS Nova 2 Omni Preview - [PR 375](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/375)
- [US-724593](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-724593): - Add usage metrics for Chat Completions streaming - [PR 371](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/371)
- [US-726288](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-726288): Add Amazon Nova 2 Pro Model Support for AWS Bedrock - [PR 370](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/370)
- [US-0009](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-0009): JAH update policyGovernanceServiceVersion - [PR 366](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/366)

### Resolved Bugs

- [BUG-971917](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-971917): Enable retry for LLM calls - [PR 380](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/380)
- [BUG-975964](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975964): -update infinity service account - [PR 392](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/392)
- [BUG-976141](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-976141): -adjust-sce-params - [PR 391](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/391)
- [BUG-974434](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-974434): - Same model_label is displaying for NovaLite - [PR 384](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/384)
- [BUG-975339](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975339): Gemini 2.0 Flash is returning as Deprecated - [PR 387](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/387)
- [BUG-971785](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-971785): Generate output_models from max_completion_tokens - [PR 369](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/369)

### Addressed Issues

- [ISSUE-139431](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-139431): Porting patch changes to main branch - [PR 378](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/378)
- [ISSUE-139208](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-139208): Add MODEL_METADATA_PATH config variable - [PR 373](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/373)

---
<br>



<a name='1.47.4-20260212153521'></a>
# [1.47.4-20260212153521 (2026-02-12)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.47.4-20260212153521)
### Resolved Bugs

- [BUG-976141](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-976141): fix params - [PR 390](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/390)

---
<br>



<a name='1.47.3-20260212084805'></a>
# [1.47.3-20260212084805 (2026-02-12)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.47.3-20260212084805)
### Resolved Bugs

- [BUG-975964](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975964): -update architecture-governance version that includes new infinity service account - [PR 389](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/389)

---
<br>



<a name='1.47.2-20260206144958'></a>
# [1.47.2-20260206144958 (2026-02-06)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.47.2-20260206144958)
### Resolved Bugs

- [BUG-975339](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975339): Gemini 2.0 Flash is marked as deprecated in Gateway - [PR 385](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/385)

---
<br>



<a name='1.47.1-20260123130738'></a>
# [1.47.1-20260123130738 (2026-01-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.47.1-20260123130738)
### Resolved Bugs

- [BUG-972680](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-972680): Allow model inference in different region for GCP - [PR 374](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/374)
- [BUG-972680](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-972680): Rename/Remove prompt from product upgrade about GCP Streaming - [PR 372](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/372)

---
<br>



<a name='1.47.0-20260120071943'></a>
# [1.47.0-20260120071943 (2026-01-20)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.47.0-20260120071943)
### New Functionality

- [US-724644-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-724644-1): GenAI Infra for GCP in Product Catalog - [PR 361](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/361)
- [US-716758](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-716758): -gateway support for gpt 5.1 and 5.2 - [PR 345](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/345)
- [US-719465-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-719465-1): Models API can show if Streaming is supported for GCP Vertex - [PR 343](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/343)
- [US-705098-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-705098-1): - Add CRIS prefix as an SCE input - [PR 344](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/344)
- [US-716954](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-716954): - Add Sonnet and Haiku 4.5 to GenAI Infra product - [PR 342](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/342)
- [US-686271-6](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-686271-6): Streaming is supported by GCP Vertex with Chat Completions API - [PR 277](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/277)

### Completed Tasks

- [TASK-1847474](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/task/TASK-1847474): - Fix Debug log print - [PR 358](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/358)

### Resolved Bugs

- [BUG-972033](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-972033): Missing input_tokens metadata for Sonnet and Haiku 4.5 - [PR 368](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/368)
- [BUG-972003](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-972003): Unused Terraform variable fails provisioning in PCFG - [PR 367](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/367)
- [BUG-967866](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-967866): Add Model description field - [PR 365](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/365)
- [BUG-970711](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-970711): - InferenceProfile upgradeValue is not declared - [PR 362](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/362)
- [BUG-971170](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-971170): [infra] DefaultSmart for PCFG should be claude-3-7-sonnet - [PR 363](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/363)
- [BUG-971169](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-971169): [infra] upgrade error - CRIS is not available for us-gov-west-1 - [PR 364](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/364)
- [BUG-965478](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-965478): - fix for invalid max output tokens for model gpt-4o - [PR 349](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/349)
- [BUG-963400](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-963400): , BUG-963401-update go and crypto version to fix vulnerabilities - [PR 353](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/353)
- [BUG-969892](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-969892): - Update API documentation - [PR 355](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/355)
- [BUG-968370](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-968370): - SelfStudyBuddy not configured is returning 500 (PCFG) - [PR 352](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/352)
- [BUG-957072](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-957072): Token usage metadata not recognzied on Converse Stream calls - [PR 348](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/348)
- [BUG-967517](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-967517): - Jenkins pipeline fails with Docker Hub rate limit error during wiremock-up target - [PR 346](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/346)
- [BUG-965197](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-965197): gpt-4o-2024-05-13 is in use bug marked as deprecated - [PR 341](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/341)

### Addressed Issues

- [ISSUE-139077](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-139077): Remove mpatch from Unit Test TestEnrichModels - [PR 359](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/359)
- [ISSUE-139087](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-139087): Mark flaky test as pending - [PR 360](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/360)
- [ISSUE-139053](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-139053): Use ClientFactory for SecretsManagerClient - [PR 356](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/356)

---
<br>



<a name='1.46.5-20251205085809'></a>
# [1.46.5-20251205085809 (2025-12-05)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.46.5-20251205085809)
### No Notable Updates Found

---
<br>



<a name='1.46.4-20251204215622'></a>
# [1.46.4-20251204215622 (2025-12-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.46.4-20251204215622)
### No Notable Updates Found

---
<br>



<a name='1.46.3-20251204150802'></a>
# [1.46.3-20251204150802 (2025-12-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.46.3-20251204150802)
### No Notable Updates Found

---
<br>



<a name='1.46.2-20251203203502'></a>
# [1.46.2-20251203203502 (2025-12-03)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.46.2-20251203203502)
### No Notable Updates Found

---
<br>



<a name='1.46.1-20251203151003'></a>
# [1.46.1-20251203151003 (2025-12-03)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.46.1-20251203151003)
### No Notable Updates Found

---
<br>



<a name='1.47.0-20251127134710'></a>
# [1.47.0-20251127134710 (2025-11-27)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.47.0-20251127134710)
### New Functionality

- [US-715361](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-715361): -integrate nova-2-lite - [PR 337](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/337)
- [US-710968](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-710968): - Separate processing time from model call time in metrics - [PR 338](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/338)

---
<br>



<a name='1.46.0-20251124183616'></a>
# [1.46.0-20251124183616 (2025-11-24)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.46.0-20251124183616)
### New Functionality

- [US-711706](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-711706): Include GPT-5 in intelligent max token framework - [PR 329](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/329)
- [US-712828](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-712828): **no merge message found** - [PR 327](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/327)
- [US-710735](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-710735): - Update models metadata to include fallback model - [PR 323](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/323)
- [US-708771-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-708771-1): - Add Sonnet and Haiku 4.5 to adaptative max token framework - [PR 320](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/320)
- [US-712290](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-712290): Support for Gpt 4.1 family - [PR 322](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/322)
- [US-710703](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-710703): gemini 2.5 flash-lite support for gateway - [PR 321](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/321)
- [US-707517](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-707517): - Add support to Claude Sonnet and Haiku 4.5 - [PR 316](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/316)

### Resolved Bugs

- [BUG-963188](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-963188): -  Error: Provider configuration not present - [PR 336](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/336)
- [BUG-961954](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-961954): - List models - [PR 335](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/335)
- [BUG-961954](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-961954): - Fix list models issues - [PR 334](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/334)
- [BUG-960297](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-960297): PCFG Bedrock FIPS endpoint is always using us-gov-east-1 - [PR 333](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/333)
- [BUG-960008](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-960008): Agent Tracer performance metrics are empty - main branch - [PR 330](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/330)
- [BUG-957630](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-957630): image params - [PR 319](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/319)
- [BUG-957072](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-957072): Response time header inconsistent for Bedrock provider - [PR 318](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/318)

### Addressed Issues

- [ISSUE-137780](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-137780): Sample prompts for testing - [PR 317](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/317)

---
<br>



<a name='1.45.1-20251104172548'></a>
# [1.45.1-20251104172548 (2025-11-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.45.1-20251104172548)
### No Notable Updates Found

---
<br>



<a name='1.45.0-20251021134245'></a>
# [1.45.0-20251021134245 (2025-10-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.45.0-20251021134245)
### New Functionality

- [US-698446](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-698446): Expose GPT-5 Models via GenAI Gateway - [PR 312](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/312)
- [US-701574-2](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-701574-2): Add response metadata header X-Genai-Gateway-Time-To-First-Token - [PR 303](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/303)
- [US-701086-3](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-701086-3): Adaptive max tokens - [PR 298](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/298)
- [US-684479](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-684479): -adding transaction related data into logs - [PR 305](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/305)
- [US-703159-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-703159-1): add gemini-2.5-pro and flash - [PR 295](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/295)
- [US-704525-2](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-704525-2): - Enhanced Model Metadata and Autopilot Compatibility - [PR 296](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/296)
- [US-694941](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-694941): Update Helm to v3.18 - [PR 300](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/300)
- [US-704525-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-704525-1): - Enhance /models endpoint to align with Autopilot List Models format - [PR 293](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/293)
- [US-701086](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-701086): Added internal/models pkg - [PR 292](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/292)

### Resolved Bugs

- [BUG-957377](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-957377): - return input_tokens and fix gpt metadata - [PR 314](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/314)
- [BUG-957357](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-957357): - change way we construct name - [PR 315](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/315)
- [BUG-957302](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-957302): -add transaction related info back to logs - [PR 313](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/313)
- [BUG-956477](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-956477): - Review of expected values by autopilot - [PR 311](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/311)
- [BUG-956477](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-956477): - Review expected values by autopilot - [PR 310](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/310)
- [BUG-955622](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955622): TextExtraction with Bedrock fails with InvalidSignature - [PR 302](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/302)
- [BUG-940840](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-940840): Update timeout to 15m for GCP generative calls (images and text) - [PR 297](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/297)
- [BUG-940840](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-940840): - Connect genAI is throwing errors randomly for file extraction - [PR 294](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/294)
- [BUG-948708](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-948708): streming headers - [PR 284](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/284)
- [BUG-950100](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-950100): [GW] GenAI Infra calculate wrong IAM permission in APAC w CRIS - [PR 288](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/288)

---
<br>



<a name='1.44.2-20251007105049'></a>
# [1.44.2-20251007105049 (2025-10-07)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.44.2-20251007105049)
### No Notable Updates Found

---
<br>



<a name='1.44.1-20251006181945'></a>
# [1.44.1-20251006181945 (2025-10-07)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.44.1-20251006181945)
### Resolved Bugs

- [BUG-955277](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955277): Product upgrade do not allow configure EnabledProvideres - [PR 299](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/299)

---
<br>



<a name='1.44.0-20250918135047'></a>
# [1.44.0-20250918135047 (2025-09-19)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.44.0-20250918135047)
### New Functionality

- [US-697955](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-697955): - Disable Specific Providers at Gateway Level - [PR 283](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/283)
- [US-699331](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-699331): Rulebase Service (Launchpad) as a GenAI Gateway upstream - [PR 281](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/281)
- [US-697954](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-697954): - Override Default Smart and Fast Model at Gateway Level - [PR 282](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/282)

### Resolved Bugs

- [BUG-952541](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-952541): **no merge message found** - [PR 290](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/290)
- [BUG-952541](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-952541): Fails to pull external-secret in GCP - cloudk-store not found - [PR 289](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/289)
- [BUG-948637](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-948637): Update GenAIHubService SCE in GCP fails because of validation - [PR 285](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/285)

---
<br>



<a name='1.43.1-20250915094120'></a>
# [1.43.1-20250915094120 (2025-09-15)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.43.1-20250915094120)
### Resolved Bugs

- [BUG-952003](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-952003): Migrate Exteranl Secret schema from v1beta1 to v1 - [PR 286](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/286)

---
<br>



<a name='1.44.0-20250825133803'></a>
# [1.44.0-20250825133803 (2025-09-02)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.44.0-20250825133803)
### New Functionality

- [US-695494](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-695494): - Add targetApi to OpenAI and GCP model configs - [PR 279](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/279)

---
<br>



<a name='1.44.0-20250731091936'></a>
# [1.44.0-20250731091936 (2025-07-31)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.44.0-20250731091936)
### New Functionality

- [US-692984](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-692984): - change secret name, ensure there are no null values in response - [PR 278](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/278)
- [US-692984](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-692984): enhance models - [PR 275](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/275)
- [US-694661](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-694661): Add support for eso v1 as v1beta1 will be deprecated in CloudK 3.42 release [merge before CloudK 3.42] - [PR 254](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/254)
- [US-686271](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-686271): Streaming is supported by GCP Vertex with Chat Completions API - [PR 276](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/276)

---
<br>



<a name='1.43.0-20250722135309'></a>
# [1.43.0-20250722135309 (2025-07-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.43.0-20250722135309)
### New Functionality

- [US-691077](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-691077): Enable Auto Mapping during upgrade for AWS fleet - [PR 274](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/274)
- [US-690274](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-690274): default smart fast - [PR 265](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/265)
- [US-691076](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-691076): Flag to phase down/replace models from GenAI Infrastructure - [PR 271](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/271)
- [US-677868](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-677868): -  Update GO version to 1.24.2 - [PR 266](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/266)
- [US-672832-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-672832-1): -make api a list, adjust api spec - [PR 263](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/263)
- [US-672832-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-672832-1): get models api - [PR 260](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/260)
- [US-686270](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-686270): - Streaming is supported by AWS Bedrock models in Gateway - [PR 262](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/262)

### Resolved Bugs

- [BUG-941296](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-941296): Add logic to define AWS partition - [PR 273](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/273)
- [BUG-940334](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-940334): AWS Bedrock CRIS ARN is calculated as us instead of us-gov - [PR 272](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/272)
- [BUG-938752](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938752): - Update GW TPS calculation to reflect Agents calculation - [PR 269](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/269)
- [BUG-937374](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-937374): GenAIHubService SCE update fail on regex validation of GenAIURL - [PR 268](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/268)
- [BUG-932003](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-932003): Add backward compatibility with embeddings API to Bedrock - [PR 267](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/267)

### Addressed Issues

- [ISSUE-134716](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-134716): Tests for converse and converse-stream in manual test suite - [PR 270](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/270)
- [ISSUE-134524](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-134524): Configure new CI cluster with CloudK 3.40 - [PR 264](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/264)

---
<br>



<a name='1.42.0-20250626140812'></a>
# [1.42.0-20250626140812 (2025-06-27)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.42.0-20250626140812)
### Resolved Bugs

- [BUG-933945](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-933945): Gateway increases TTFT by buffering tokens while using streaming - [PR 261](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/261)

---
<br>



<a name='1.41.0-20250623160130'></a>
# [1.41.0-20250623160130 (2025-06-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.41.0-20250623160130)
### New Functionality

- [US-688223](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-688223): - Change Bedrock endpoint configuration - [PR 252](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/252)

### Resolved Bugs

- [BUG-933345](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-933345): Expose resource limits and requests for service container as SCE inputs  (cont) - [PR 259](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/259)
- [BUG-933345](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-933345): - Expose resource limits and requests for service container as SCE inputs - [PR 258](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/258)
- [BUG-934790](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-934790): Exception error on gateway passing buddy response back - [PR 256](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/256)
- [BUG-933876](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-933876): Fix content length handling in gzip plugin for requests and responses - [PR 255](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/255)

---
<br>



<a name='1.40.0-20250613090420'></a>
# [1.40.0-20250613090420 (2025-06-13)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.40.0-20250613090420)
### New Functionality

- [US-688038](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-688038): Amazon Nova models can be deployed on demand - [PR 251](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/251)
- [US-687231](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-687231): - GenAI Infrastructure ready for adding streaming as target API - [PR 250](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/250)
- [US-684482](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-684482): Add tokens per second metric and improve tracer initialization - [PR 249](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/249)
- [US-684483](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-684483): Support compressed payload when introspecting response metrics - [PR 246](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/246)
- [US-685277](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-685277): -Deployed new cluster for the Integration testing - [PR 245](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/245)
- [US-654663-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-654663-1): -upgraded the helmCLI version to 61.2.0 - [PR 242](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/242)

### Resolved Bugs

- [BUG-929235](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-929235): Fix TPS calculation and add retry-count in Gateway HTTP headers - [PR 247](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/247)

### Addressed Issues

- [ISSUE-133669](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-133669): Semi automated tool to support version testing - [PR 241](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/241)

---
<br>



<a name='1.39.0-20250522183114'></a>
# [1.39.0-20250522183114 (2025-05-22)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.39.0-20250522183114)
### New Functionality

- [US-680370](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-680370): GenAI Gateway sends response headers about LLM performance - [PR 244](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/244)

---
<br>



<a name='1.38.0-20250521011755'></a>
# [1.38.0-20250521011755 (2025-05-21)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.38.0-20250521011755)
### New Functionality

- [US-682677](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-682677): Record in logs metadata from GenAI Model calls and more information logs - [PR 240](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/240)
- [US-683180](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-683180): Fix gateway OTEL instrumentation to not loose traces - [PR 238](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/238)
- [US-670574](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-670574): GenAI Infra model mappings are updated in gateway without product update - [PR 236](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/236)

### Resolved Bugs

- [BUG-926628](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-926628): Fixed the input & output token Metrics - [PR 235](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/235)

---
<br>



<a name='1.37.0-20250514190732'></a>
# [1.37.0-20250514190732 (2025-05-14)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.37.0-20250514190732)
### New Functionality

- [US-683180](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-683180): Adding OTLP default settings tracing to Gateway - [PR 237](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/237)

### Resolved Bugs

- [BUG-921547](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-921547): -Titan Text Embed model(Lambda) should work for Vector store - [PR 232](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/232)

---
<br>



<a name='1.36.0-20250404074152'></a>
# [1.36.0-20250404074152 (2025-04-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.36.0-20250404074152)
### New Functionality

- [US-674336](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-674336): -Fedramp support - GenAI Gateway uses FIPS endpoint for Bedrock - [PR 228](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/228)
- [US-675245](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-675245): Manage Go tooling with Go Tool command - [PR 229](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/229)

### Addressed Issues

- [ISSUE-131289](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-131289): updated the GO version from 1.22.8 to 1.24.1 - [PR 227](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/227)

---
<br>



<a name='1.35.0-20250328183549'></a>
# [1.35.0-20250328183549 (2025-03-28)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.35.0-20250328183549)
### New Functionality

- [US-670920](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-670920): Fedramp GenAI Gateway and Infra to work in AWS-US-GOV partition - [PR 226](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/226)
- [US-670345](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-670345): making the ServiceAuthenticationClientService SCE a dependency for the GenAIHubService SCE - [PR 216](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/216)
- [US-659075](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-659075): Gateway supports Launchpad with GenAI Infrastructure - [PR 219](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/219)
- [US-669876](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-669876): Creating a new OIDC assumable role to list and read AWS Secrets - [PR 209](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/209)

### Resolved Bugs

- [BUG-918414](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-918414): Sax Registration SCE fail to be registered in PCFG Control Plane - [PR 225](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/225)

### Addressed Issues

- [ISSUE-132107](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-132107): Delete ResourceGUID created by Integration Test suite - [PR 215](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/215)

---
<br>



<a name='1.34.0-20250312163155'></a>
# [1.34.0-20250312163155 (2025-03-12)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.34.0-20250312163155)
### New Functionality

- [US-671172](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-671172): adding param for claude 3.7 sonnet - [PR 212](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/212)
- [US-669401](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-669401): Make claude-3.5-sonnet-v2 to allowed models with option to use Reginal Inference Profile - [PR 204](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/204)
- [US-669876](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-669876): Set the ops container version to nonroot - [PR 210](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/210)
- [US-663602](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-663602): -GCP-VertexAI-NewModels - [PR 208](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/208)
- [US-664923-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-664923-1): created a separate container image for the genai-ops deployment - [PR 201](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/201)
- [US-669878](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-669878): Change default version of GPT-4o from 2024 05 13 to 2024 08 06 - [PR 207](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/207)
- [US-669896](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-669896): Made changes for the Model Mapping AWS Secret to be created in the AWS LLM Account - [PR 206](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/206)
- [US-668590](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-668590): add support for titan-embed-text-v2 managed via GenAI Infrastructure - [PR 200](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/200)
- [US-668373](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-668373): Remove instrumentation that was used for bedrock sdk - [PR 199](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/199)

### Resolved Bugs

- [BUG-916139](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-916139): GenAIHubService param typo on External Secret helm - [PR 213](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/213)

### Addressed Issues

- [ISSUE-130853](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-130853): Use HTTP call against AWS Bedrock Service endpoint directly instead of using AWS Bedrock SDK - [PR 196](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/196)

---
<br>



<a name='1.33.0-20250221150447'></a>
# [1.33.0-20250221150447 (2025-02-21)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.33.0-20250221150447)
### New Functionality

- [US-662068](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-662068): Update Helm to 3.16.3 - [PR 197](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/197)
- [US-664542](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-664542): titan-text-embedding-v2 - [PR 195](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/195)
- [US-665904](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-665904): GenAIGatewayService titan support - [PR 194](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/194)

---
<br>



<a name='1.32.0-20250207204434'></a>
# [1.32.0-20250207204434 (2025-02-07)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.32.0-20250207204434)
### New Functionality

- [US-592521](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-592521): Add metrics collection for all providers - [PR 192](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/192)

---
<br>



<a name='1.31.0-20250117185103'></a>
# [1.31.0-20250117185103 (2025-01-17)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.31.0-20250117185103)
### Resolved Bugs

- [BUG-906482](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-906482): fix parsing of pe access role for sax iam oidc provider - [PR 190](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/190)

---
<br>



<a name='1.30.0-20250117164618'></a>
# [1.30.0-20250117164618 (2025-01-17)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.30.0-20250117164618)
### Resolved Bugs

- [BUG-906482](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-906482): fix parsing of pe access role - [PR 189](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/189)

---
<br>



<a name='1.29.0-20250113183340'></a>
# [1.29.0-20250113183340 (2025-01-13)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.29.0-20250113183340)
### New Functionality

- [US-659987](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-659987): Add new histogram buckets for response time metric - [PR 186](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/186)
- [US-622947](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622947): Redefine the buckets for response time histogram - [PR 184](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/184)

### Resolved Bugs

- [BUG-905536](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-905536): Enable prometheus metrics scrapping to the endpoint /metrics - [PR 183](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/183)

---
<br>



<a name='1.28.0-20241231110610'></a>
# [1.28.0-20241231110610 (2025-01-03)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.28.0-20241231110610)
### New Functionality

- [US-622952](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622952): add AWS Region information needed for creating an AWS Config - [PR 182](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/182)
- [US-622952](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622952): simplify SCE output to facilitate operation - [PR 181](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/181)
- [US-643570-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-643570-1): Added support for Claude-3-Haiku provisioned via GenAI Infra Control Plane SCE - [PR 180](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/180)

---
<br>



<a name='1.27.0-20241220104621'></a>
# [1.27.0-20241220104621 (2024-12-20)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.27.0-20241220104621)
### New Functionality

- [US-651731](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-651731): -upgraded the ServiceAuthenticationClientService - [PR 175](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/175)

### Resolved Bugs

- [BUG-901938](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-901938): Price spike by ListSecrets calls to AWS after adding PrivateModels to GenAI Gateway - [PR 179](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/179)

---
<br>



<a name='1.26.0-20241216103103'></a>
# [1.26.0-20241216103103 (2024-12-16)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.26.0-20241216103103)
### New Functionality

- [US-654801](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-654801): increasing the deadline for the imagen backend to 5 mins - [PR 176](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/176)

---
<br>



<a name='1.26.0-20241206133719'></a>
# [1.26.0-20241206133719 (2024-12-13)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.26.0-20241206133719)
### New Functionality

- [US-652516](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-652516): -Replaced gemini-1-5-pro-preview with gemini-1-5-pro & gemini-1-5-flash models - [PR 173](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/173)

---
<br>



<a name='1.25.0-20241203174748'></a>
# [1.25.0-20241203174748 (2024-12-05)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.25.0-20241203174748)
### New Functionality

- [US-653767](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-653767): update test cluster GUID for integration tests - [PR 171](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/171)
- [US-647488](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-647488): -Replaced the support of GenerateContent by OpenAI API for calling GCP VertexAI Gemini model - [PR 166](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/166)
- [US-647093-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-647093-1): VertexAI Cloud Function reads project-id and location from GCP metadata API - [PR 168](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/168)
- [US-647093-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-647093-1): adding support for GCP imagen - [PR 165](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/165)
- [US-620624-7](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-620624-7): HTTP metrics for models - [PR 163](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/163)
- [US-647491](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-647491): update gpt-4o-next with new gpt-4o model - [PR 159](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/159)

---
<br>



<a name='1.24.0-20241119092112'></a>
# [1.24.0-20241119092112 (2024-11-19)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.24.0-20241119092112)
### New Functionality

- [US-632581](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-632581): MRDR relatedupdates - [PR 157](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/157)
- [US-649218](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-649218): Add treatment to add inference profile to modelId in AWS Bedrock request payload - [PR 154](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/154)
- [US-632581](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-632581): MRDR - [PR 152](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/152)
- [US-649218](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-649218): Add support for claude-3-5-haiku and claude-3-5-sonnet - [PR 153](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/153)

---
<br>



<a name='1.23.0-20241030133716'></a>
# [1.23.0-20241030133716 (2024-10-30)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.23.0-20241030133716)
### New Functionality

- [US-631593-8](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-631593-8): making minor fixes - [PR 149](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/149)
- [US-648055](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-648055): adding gpt-4o-mini support for Private Models - [PR 148](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/148)

---
<br>



<a name='1.22.0-20241021152240'></a>
# [1.22.0-20241021152240 (2024-10-21)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.22.0-20241021152240)
### New Functionality

- [US-644432](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-644432): Enable creation of ingress configuration to the ops endpoint - [PR 143](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/143)
- [US-644432](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-644432): Adds Ops pod to GenAI Gateway Service to compute GenAI events and provide Metrics to GOC - [PR 141](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/141)

### Resolved Bugs

- [BUG-893223](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-893223): Fix chartName to override directory name to fix helm upgrade - [PR 142](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/142)

---
<br>



<a name='1.21.0-20240930102757'></a>
# [1.21.0-20240930102757 (2024-09-30)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.21.0-20240930102757)
### New Functionality

- [US-622946](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622946): GenAI Infrastructue - AWS Bedrock - remove unecessary filter in SCE - [PR 136](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/136)
- [US-622946](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622946): GenAI Infrastructue - AWS Bedrock - [PR 135](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/135)

### Resolved Bugs

- [BUG-890342](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-890342): Keep Control Plane AWS provider in the default region - [PR 137](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/137)

---
<br>



<a name='1.21.0-20240923062644'></a>
# [1.21.0-20240923062644 (2024-09-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.21.0-20240923062644)
### New Functionality

- [US-640307](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-640307): changelogplugin - [PR 134](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/134)

---
<br>



<a name='1.20.0-20240829145911'></a>
# [1.20.0-20240829145911 (2024-08-29)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.20.0-20240829145911)
### Resolved Bugs

- [BUG-885279](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-885279): - Fix integration between Smart Chunking and Gateway - [PR 127](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/127)

---
<br>



<a name='1.19.0-20240828104506'></a>
# [1.19.0-20240828104506 (2024-08-28)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.19.0-20240828104506)
### Resolved Bugs

- [BUG-883590](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-883590): Remove the secret fetching from HubService Pod definition - [PR 126](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/126)

---
<br>



<a name='1.18.0-20240814122626'></a>
# [1.18.0-20240814122626 (2024-08-14)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.18.0-20240814122626)
### New Functionality

- [US-622543-3](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622543-3): Introducing the AWS bedrock converse API to handle all AWS-bedrock backed model calls - [PR 116](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/116)

---
<br>



<a name='1.17.0-20240812114648'></a>
# [1.17.0-20240812114648 (2024-08-12)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.17.0-20240812114648)
### New Functionality

- [US-630949](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-630949): -change defaultValues for GenAIURL - [PR 121](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/121)
- [US-623889](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-623889): GenAIBYOMMapping SCE - [PR 118](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/118)

---
<br>



<a name='1.16.0-20240809111303'></a>
# [1.16.0-20240809111303 (2024-08-09)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.16.0-20240809111303)
### New Functionality

- [US-628977](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-628977): Updated the service base version to 1.14.0-20240603204538 - [PR 111](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/111)

### Resolved Bugs

- [BUG-881307](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-881307): -change UseSax expr - [PR 119](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/119)

### Addressed Issues

- [ISSUE-126744](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-126744): Updating Release Notes - [PR 115](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/115)

---
<br>



<a name='1.15.0-20240802133716'></a>
# [1.15.0-20240802133716 (2024-08-02)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.15.0-20240802133716)
### New Functionality

- [US-629317](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-629317): add gpt-4o-mini - [PR 112](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/112)

---
<br>



<a name='1.14.0-20240723133719'></a>
# [1.14.0-20240723133719 (2024-07-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.14.0-20240723133719)
### No Notable Updates Found

---
<br>



<a name='1.13.0-20240606123115'></a>
# [1.13.0-20240606123115 (2024-06-06)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.13.0-20240606123115)
### No Notable Updates Found

---
<br>



<a name='1.12.0-20240515082151'></a>
# [1.12.0-20240515082151 (2024-05-15)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.12.0-20240515082151)
### No Notable Updates Found

---
<br>



<a name='1.11.0-20240430184018'></a>
# [1.11.0-20240430184018 (2024-04-30)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.11.0-20240430184018)
### New Functionality

- [US-613104](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-613104): GCP Vertex AI model experimental support - [PR 58](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/58)
- [US-612513](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-612513): changing genai dev cluster - [PR 55](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/55)
- [US-611174-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-611174-1): Lambda code to invoke Claude 3 Haiku model - [PR 54](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/54)

### Resolved Bugs

- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): Added missed endpoint - [PR 57](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/57)
- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): Make feature AvoidCopyrightInfringements configurable - [PR 56](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/56)
- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): broken request - [PR 53](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/53)

### Addressed Issues

- [ISSUE-123830](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-123830): Added support of LOG_LEVEL env variable - [PR 52](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/52)

---
<br>



<a name='1.11.0-20240430181511'></a>
# [1.11.0-20240430181511 (2024-04-30)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.11.0-20240430181511)
### No Notable Updates Found

---
<br>



<a name='1.10.0-20240426114441'></a>
# [1.10.0-20240426114441 (2024-04-26)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.10.0-20240426114441)
### New Functionality

- [US-613104](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-613104): GCP Vertex AI model experimental support - [PR 58](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/58)
- [US-612513](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-612513): changing genai dev cluster - [PR 55](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/55)
- [US-611174-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-611174-1): Lambda code to invoke Claude 3 Haiku model - [PR 54](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/54)

### Resolved Bugs

- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): Added missed endpoint - [PR 57](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/57)
- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): Make feature AvoidCopyrightInfringements configurable - [PR 56](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/56)
- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): broken request - [PR 53](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/53)

### Addressed Issues

- [ISSUE-123830](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-123830): Added support of LOG_LEVEL env variable - [PR 52](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/52)

---
<br>



<a name='1.9.0-20240425170532'></a>
# [1.9.0-20240425170532 (2024-04-25)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.9.0-20240425170532)
### New Functionality

- [US-612513](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-612513): changing genai dev cluster - [PR 55](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/55)
- [US-611174-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-611174-1): Lambda code to invoke Claude 3 Haiku model - [PR 54](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/54)

### Resolved Bugs

- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): Added missed endpoint - [PR 57](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/57)
- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): Make feature AvoidCopyrightInfringements configurable - [PR 56](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/56)
- [BUG-865978](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-865978): broken request - [PR 53](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/53)

### Addressed Issues

- [ISSUE-123830](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-123830): Added support of LOG_LEVEL env variable - [PR 52](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/52)

---
<br>



<a name='1.8.0-20240424085523'></a>
# [1.8.0-20240424085523 (2024-04-24)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.8.0-20240424085523)
### No Notable Updates Found

---
<br>



<a name='1.7.0-20240418142321'></a>
# [1.7.0-20240418142321 (2024-04-18)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.7.0-20240418142321)
### No Notable Updates Found

---
<br>



<a name='1.6.0-20240313071300'></a>
# [1.6.0-20240313071300 (2024-03-13)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.6.0-20240313071300)
### New Functionality

- [US-600847](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-600847): Externalize models.yaml file - [PR 41](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/41)
- [US-600847](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-600847): buddySupport - [PR 40](https://git.pega.io/projects/PCLD/repos/gen-ai-hub-service/pull-requests/40)

---
<br>



<a name='1.5.0-20240306162912'></a>
# [1.5.0-20240306162912 (2024-03-06)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.5.0-20240306162912)
### No Notable Updates Found

---
<br>



<a name='1.4.0-20240306102134'></a>
# [1.4.0-20240306102134 (2024-03-06)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.4.0-20240306102134)
### No Notable Updates Found

---
<br>



<a name='1.3.0-20240304085218'></a>
# [1.3.0-20240304085218 (2024-03-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.3.0-20240304085218)
### No Notable Updates Found

---
<br>



<a name='1.2.0-20231103151127'></a>
# [1.2.0-20231103151127 (2023-11-03)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.2.0-20231103151127)
### No Notable Updates Found

---
<br>



<a name='1.0.0-20231011111033'></a>
# [1.0.0-20231011111033 (2023-10-11)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-hub-service%2FGenaiHubService%2F1.0.0-20231011111033)
### No Notable Updates Found

---
<br>





