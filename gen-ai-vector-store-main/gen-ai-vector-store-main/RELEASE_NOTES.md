
# Changelog

---

<br>

<Unreleased>

<a name='0.22.1-20260417071504'></a>
### [0.22.1-20260417071504 (2026-04-17)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.22.1-20260417071504)
### Resolved Bugs

- [BUG-986839](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-986839): document search overscan - [PR 746](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/746)
- [BUG-986916](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-986916): Fix DB pool exhaustion cascade in async document processing - [PR 751](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/751)
- [BUG-987760](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-987760): Upgrade go-sax to v1.3.11 to fix nil pointer crash - [PR 754](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/754)

---
<br>



<a name='0.22.0-20260409135518'></a>
### [0.22.0-20260409135518 (2026-04-09)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.22.0-20260409135518)
### New Functionality

- [US-732053](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-732053): Mark GenAISmartChunking and GenAISmartChunkingRole SCEs with `action: DELETE` - [PR 692](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/692)
- [US-722568](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-722568): Integration Testsuites migration - [PR 693](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/693)
- [US-732092](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-732092): Remove GenAISmartChunking and GenAISmartChunkingRole SCE - [PR 695](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/695)
- [US-734021](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-734021): Add missing release notes entries after GitHub migration - [PR 696](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/696)
- [US-733705](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-733705): Added processing overhead headers - [PR 700](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/700)
- [US-736080](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-736080): Add AI tooling configuration and development skills - [PR 703](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/703)
- [US-737303](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-737303): DB Size Endpoint - [PR 704](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/704)
- [US-732361](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-732361): Add new smart chunking service accounts and network connectivity - [PR 709](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/709)
- [US-732363-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-732363-1): Modify /file and /file-text endpoints to call SC /job API - [PR 711](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/711)
- [US-711945](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-711945): Remove DBQSUrl override from VS product catalog - [PR 714](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/714)
- [US-739872](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-739872): Add Claude Code agent team + GitHub Copilot AGENTS.md - [PR 715](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/715)
- [US-732363-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-732363-1): Add default fallback URL for smart chunking SCE parameter - [PR 716](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/716)
- [US-731279](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-731279): Add smart attribution flags and restructure file upload metadata API - [PR 717](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/717)
- [US-739872](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-739872): Add Copilot Coding Agent setup with 10 VS-specific custom agents - [PR 726](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/726)
- [US-743272](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-743272): Accept empty docs - [PR 731](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/731)
- [US-743904](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-743904): Add GCP Isolation and Role Terraform EDR parity with AWS - [PR 732](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/732)
- [US-744227](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-744227): nullable params - [PR 741](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/741)

### Resolved Bugs

- [BUG-984996](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-984996): Remove file extension validation from /file endpoint - [PR 718](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/718)
- [BUG-985025](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-985025): Coerce nil attribute values to empty slice before sending to Smart Chunking - [PR 720](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/720)
- [BUG-984996](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-984996): Remove file extension validation integration tests - [PR 721](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/721)

---
<br>



<a name='0.21.0-20260222210840'></a>
### [0.21.0-20260222210840 (2026-02-22)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.21.0-20260222210840)
### New Functionality

- [US-714331](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-714331): token caching - [PR 622](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/622)
- [US-609740-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-609740-1): add new db metrics - [PR 654](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/654)
- [US-718620-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-718620-1): added a new metric vector_store_db_attribute_value_count - [PR 656](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/656)
- [US-721083](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-721083): PDCEndpoint value in GenAIVectorStoreIsolation - [PR 658](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/658)
- [US-722289](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-722289): DbInstanceSCE version upgrade - [PR 659](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/659)
- [US-695558](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-695558): Changelog plugin version update - [PR 660](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/660)
- [US-588080-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-588080-1): Add consumer contract tests for GenAI Gateway service - [PR 661](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/661)
- [US-721095](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-721095): push semantic search metrics to PDC - [PR 666](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/666)
- [US-721097](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-721097): push db metrics to PDC - [PR 670](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/670)
- [US-679746](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-679746): Improved VS integration test helper functions - [PR 672](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/672)
- [US-725900](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-725900): Automatic ANALYZE after PostgreSQL Version Upgrade - [PR 673](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/673)
- [US-726519](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-726519): Upgrade build tooling dependencies - [PR 674](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/674)
- [US-726583](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-726583): update module and dependencies - [PR 683](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/683)
- [US-727893](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-727893): Bump gradle-cloud-services-plugins to 5.3.4 - [PR 691](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/691)
- [US-695554](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-695554): GitHub Migration - Enable GitHub Pages (Swagger UI) - [PR 634](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/634)
- [US-695554](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-695554): GitHub migration: Onboard Repository with Dependabot - [PR 633](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/633)

### Resolved Bugs

- [BUG-966570](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-966570): 0x20x0 release issues - [PR 625](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/625)
- [BUG-966560](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-966560): Removed isolation validation from ops APIs - [PR 627](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/627)
- [BUG-957178](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-957178): filter documents by embeddings' attributes - [PR 664](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/664)
- [BUG-972921](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-972921): Disable isolation ID verification by default - [PR 665](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/665)
- [BUG-955712](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955712): Normalized metrics "path" label - [PR 675](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/675)
- [BUG-955443](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955443): Handle Document Deletion During Async Processing - [PR 676](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/676)
- [BUG-975989](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975989): Allowlist the new Infinity ServiceAccount name - [PR 679](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/679)
- [BUG-975891](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975891): duplicate output value - [PR 687](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/687)

---
<br>



<a name='0.18.3-20260212153158'></a>
### [0.18.3-20260212153158 (2026-02-12)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.18.3-20260212153158)
### New Functionality

- [US-719016](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-719016): Updated EncourageSemSearchIndexUse default value to false - [PR 618](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/618)

### Resolved Bugs

- [BUG-975989](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975989): Allowlist new Infinity ServiceAccount name - [PR 678](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/678)
- [BUG-975989](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975989): Fixed SCE parameters validation - [PR 681](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/681)

---
<br>



<a name='0.20.3-20260211203407'></a>
### [0.20.3-20260211203407 (2026-02-11)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.20.3-20260211203407)
### Resolved Bugs

- [BUG-975989](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975989): Allowlist the new Infinity ServiceAccount name - [PR 677](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/677)
- [BUG-975989](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-975989): fix input parameters - [PR 680](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/680)

---
<br>



<a name='0.20.2-20260123131250'></a>
### [0.20.2-20260123131250 (2026-01-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.20.2-20260123131250)
### Resolved Bugs

- [BUG-972913](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-972913): Disable isolationID verification - [PR 662](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/662)
- [BUG-972913](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-972913): Update changelog plugin - [PR 663](https://github.com/pega-cloudengineering/gen-ai-vector-store/pull/663)

---
<br>



<a name='0.17.3-20251216133601'></a>
# [0.17.3-20251216133601 (2025-12-17)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.17.3-20251216133601)
### Resolved Bugs

- [BUG-967033](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-967033): Upgrade DBInstance SCE to 5.29.0-20250909110554 - [PR 628](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/628)

---
<br>



<a name='0.20.1-20251215084451'></a>
# [0.20.1-20251215084451 (2025-12-15)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.20.1-20251215084451)
### Resolved Bugs

- [BUG-966560](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-966560): Removed isolation validation from ops APIs - [PR 626](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/626)
- [BUG-966570](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-966570): 0x20x0 release issues - [PR 624](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/624)

---
<br>



<a name='0.20.0-20251208201652'></a>
# [0.20.0-20251208201652 (2025-12-08)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.20.0-20251208201652)
### New Functionality

- [US-609767](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-609767): add guid/isolationID verification to the requests - [PR 616](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/616)
- [US-710966](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-710966): Comment update on SAX Ops cell resolution for EDR support - [PR 609](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/609)
- [US-714933](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-714933): Re-enablement of EDR should be working - [PR 607](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/607)

### Resolved Bugs

- [BUG-956700](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-956700): DB tools installed on PCFG - [PR 621](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/621)
- [BUG-963703](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-963703): -Vulnerability in golang.org/x/crypto are fixed - [PR 620](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/620)
- [BUG-955532](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955532): merged 2 - [PR 617](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/617)
- [BUG-955683](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955683): Changed retry logic and defaults on query chynks - [PR 614](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/614)
- [BUG-962737](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-962737): for GCP reuse existing input for this param instead of using hardcoded PG14 - [PR 613](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/613)
- [BUG-943967](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-943967): ingestion db pool blocking basic - [PR 608](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/608)

---
<br>



<a name='0.19.2-20251121121616'></a>
# [0.19.2-20251121121616 (2025-11-21)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.19.2-20251121121616)
### Resolved Bugs

- [BUG-962737](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-962737): for GCP reuse existing input for this param instead of using hardcoded PG14 - [PR 612](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/612)

---
<br>



<a name='0.19.1-20251120113335'></a>
# [0.19.1-20251120113335 (2025-11-20)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.19.1-20251120113335)
### New Functionality

- [US-716090](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-716090): -mrdr re-enablement is working - [PR 610](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/610)

---
<br>



<a name='0.19.0-20251113131544'></a>
# [0.19.0-20251113131544 (2025-11-18)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.19.0-20251113131544)
### New Functionality

- [US-708287](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-708287): -Updated serviceAuthenticationClientServiceVersion  and simplify databaseSecret retrieval logic - [PR 601](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/601)
- [US-704405](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-704405): jsonb attributes 4 - [PR 600](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/600)
- [US-704405](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-704405): Refactored K6 tests - [PR 595](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/595)

### Resolved Bugs

- [BUG-946824](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-946824): -Increased MaxChunkContentSize from 8000 to 16000 - [PR 606](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/606)
- [BUG-943967](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-943967): Unblock Semantic search during bulk ingestion - [PR 585](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/585)
- [BUG-956909](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-956909): fix cloning - [PR 598](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/598)

---
<br>



<a name='0.18.2-20251106151431'></a>
# [0.18.2-20251106151431 (2025-11-07)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.18.2-20251106151431)
### New Functionality

- [US-713536](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-713536): -Updated serviceAuthenticationClientServiceVersion and simplified databaseSecret retrieval logic - [PR 603](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/603)

---
<br>



<a name='0.18.1-20251023115535'></a>
# [0.18.1-20251023115535 (2025-10-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.18.1-20251023115535)
### Resolved Bugs

- [BUG-956909](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-956909): fix cloning - [PR 597](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/597)

---
<br>



<a name='0.17.2-20251023074204'></a>
# [0.17.2-20251023074204 (2025-10-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.17.2-20251023074204)
### Resolved Bugs

- [BUG-956909](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-956909): fix cloning - [PR 596](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/596)

---
<br>



<a name='0.18.0-20251014150407'></a>
# [0.18.0-20251014150407 (2025-10-15)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.18.0-20251014150407)
### New Functionality

- [US-707376](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-707376): Prepare 0.18.0 release - [PR 592](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/592)
- [US-706156](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-706156): Changes for EmbeddingModel param - [PR 584](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/584)
- [US-695187](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-695187): updated helm version to 3.18.0 - [PR 580](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/580)
- [US-700905](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-700905): Changes to update to PG17 - [PR 578](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/578)

### Resolved Bugs

- [BUG-955555](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955555): Add support for Malaysia ap-southeast-5 region - [PR 590](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/590)
- [BUG-955456](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955456): - reduce calls for internal db metrics collection - [PR 588](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/588)

---
<br>



<a name='0.17.1-20251010163017'></a>
# [0.17.1-20251010163017 (2025-10-13)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.17.1-20251010163017)
### Resolved Bugs

- [BUG-955555](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955555): Add support for Malaysia region - [PR 591](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/591)
- [BUG-955456](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-955456): - reduce calls for internal db metrics collection - [PR 589](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/589)

---
<br>



<a name='0.17.0-20250915070122'></a>
# [0.17.0-20250915070122 (2025-09-15)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.17.0-20250915070122)
### New Functionality

- [US-702194](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-702194): Add new metrics to Grafana - [PR 576](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/576)
- [US-700610](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-700610): Create mocked up Vector Store - [PR 566](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/566)
- [US-701081](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-701081): Added context information to logs, added metrics logging, unified logger usage - [PR 572](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/572)
- [US-700921](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-700921): Moved ReadOnly mode implementation to middleware - [PR 571](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/571)
- [US-699141](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-699141): metadata headers:  Count documents and vectors - [PR 570](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/570)
- [US-699141](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-699141): metadata headers 2 - [PR 567](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/567)
- [US-695652](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-695652): Added fixing attrs2 index into migration to 0.17.0 - [PR 541](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/541)
- [US-682858-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-682858-1): process ingestion in background using separate tables - [PR 522](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/522)
- [US-694708](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-694708): -Reviewed and changed the log levels for PII logs - [PR 534](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/534)
- [US-694674](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-694674): Add troubleshooting mode - [PR 530](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/530)

### Resolved Bugs

- [BUG-938407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938407): -fixed-memory-leak - [PR 577](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/577)
- [BUG-938407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938407): - Fixed Nil Pointer Dereference - [PR 575](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/575)
- [BUG-947379](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-947379): Fixed SCE pramaters - [PR 564](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/564)
- [BUG-940008](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-940008): -If Ingestion is failed, Error messages will be user friendly - [PR 562](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/562)
- [BUG-947379](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-947379): embedding client timeout 4 merge to main 2 - [PR 561](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/561)
- [BUG-937329](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-937329): performance improvement - [PR 550](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/550)
- [BUG-943377](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-943377): -Queue processor includes attributes while trying to re-embed a chunk - [PR 549](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/549)
- [BUG-938407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938407): Fixed memory leak by creating one global logger instance - [PR 545](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/545)
- [BUG-945259](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-945259): fix response headers - [PR 544](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/544)
- [BUG-939134](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-939134): fix filters not working - [PR 539](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/539)
- [BUG-943362](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-943362): prevent background from restarts, fix invalid indexes - [PR 538](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/538)
- [BUG-943350](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-943350): -Find-documents response structure is modified - [PR 537](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/537)
- [BUG-938407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938407): -Added memory monitoring & garbage collector - [PR 531](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/531)

### Addressed Issues

- [ISSUE-135030](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-135030): add chunk sorting back - [PR 542](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/542)

---
<br>



<a name='0.16.5-20250903190622'></a>
# [0.16.5-20250903190622 (2025-09-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.16.5-20250903190622)
### Resolved Bugs

- [BUG-948738](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-948738): Fixed response headers - [PR 568](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/568)

---
<br>



<a name='0.16.4-20250820204944'></a>
# [0.16.4-20250820204944 (2025-08-21)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.16.4-20250820204944)
### Resolved Bugs

- [BUG-947379](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-947379): fix retries typo 016 - [PR 565](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/565)
- [BUG-947379](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-947379): Fixed SCE pramaters - [PR 563](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/563)
- [BUG-947379](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-947379): Fixed SCE default values - [PR 560](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/560)
- [BUG-945259](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-945259): fix response headers - [PR 557](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/557)
- [BUG-937329](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-937329): Calculate MD5 hashed on service pods - [PR 556](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/556)
- [BUG-947379](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-947379): embedding client timeout 4 - [PR 555](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/555)
- [BUG-947210](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-947210): Removed logging overhead on non-DEBUG levels - [PR 553](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/553)

---
<br>



<a name='0.16.3-20250809142819'></a>
# [0.16.3-20250809142819 (2025-08-09)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.16.3-20250809142819)
### Resolved Bugs

- [BUG-937329](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-937329): Improve performance, Added cache, Added/Renamed metrics - [PR 547](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/547)
- [BUG-938407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938407): Fixed memory leak by creating one global logger instance - [PR 546](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/546)

---
<br>



<a name='0.16.2-20250731093643'></a>
# [0.16.2-20250731093643 (2025-07-31)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.16.2-20250731093643)
### Resolved Bugs

- [BUG-943362](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-943362): prevent background from restarts, fix invalid indexes - [PR 536](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/536)

---
<br>



<a name='0.16.1-20250723073518'></a>
# [0.16.1-20250723073518 (2025-07-28)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.16.1-20250723073518)
### Resolved Bugs

- [BUG-938407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938407): Added memory monitoring & garbage collector - [PR 533](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/533)
- [BUG-938407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-938407): Add troubleshooting mode - [PR 532](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/532)

---
<br>



<a name='0.16.0-20250718121413'></a>
# [0.16.0-20250718121413 (2025-07-18)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.16.0-20250718121413)
### New Functionality

- [US-624382](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-624382): fix non-blocking HNSW index creation in migration - [PR 528](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/528)
- [US-624382](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-624382): index should not block search - [PR 527](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/527)
- [US-624382](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-624382): Removed calculation of create index params based on service machine - [PR 526](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/526)
- [US-624382](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-624382): Set current and minimal schema version to 0.16.0 - [PR 525](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/525)
- [US-624382](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-624382): introduced HNSW vector index to improve semantic search performace - [PR 523](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/523)

### Addressed Issues

- [ISSUE-134507](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-134507): Added Document Attributes to APIv2 Response - [PR 524](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/524)

---
<br>



<a name='0.15.0-20250602114159'></a>
# [0.15.0-20250602114159 (2025-06-13)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.15.0-20250602114159)
### New Functionality

- [US-686018](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-686018): Temporary disabled plugin - [PR 520](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/520)
- [US-686018](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-686018): Updated SDEA plugins - [PR 519](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/519)
- [US-677872](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-677872): updated golang version - [PR 517](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/517)
- [US-685698](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-685698): Indexer refactoring - [PR 516](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/516)
- [US-668632](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-668632): Dropped EmbeddingsReschedulerHandler - [PR 515](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/515)
- [US-622990](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622990): otlp - [PR 512](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/512)
- [US-675050](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-675050): - Not add Autoresloved Attributes - [PR 497](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/497)
- [US-683178](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-683178): Adding OTLP default settings tracing to Vector Store - [PR 503](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/503)
- [US-630133](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-630133): semantic search metrics - [PR 500](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/500)
- [US-679749](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-679749): ocr - [PR 498](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/498)
- [US-654789](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-654789): add embedder google text multilingual embedding 002 - [PR 495](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/495)
- [US-645214](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-645214): - Support to return attributes on vectorstore document status API - [PR 491](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/491)
- [US-675722](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-675722): move get collections processing to db - [PR 492](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/492)
- [US-670246](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-670246): uncoment file text - [PR 487](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/487)
- [US-647581](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-647581): Removed not used tool - [PR 485](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/485)

### Resolved Bugs

- [BUG-929898](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-929898): - Resource punkt_tab not found - [PR 521](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/521)
- [BUG-929654](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-929654): - Pods restarting due to health check timeout - [PR 514](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/514)
- [BUG-927944](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-927944): - Review content type handling - [PR 509](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/509)
- [BUG-922652](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-922652): change file status handling in PutDocumentFile - [PR 494](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/494)
- [BUG-922750](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-922750): Fixed isolation not found error - [PR 493](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/493)
- [BUG-922256](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-922256): Fixed schema_info function - [PR 489](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/489)

---
<br>



<a name='0.14.5-20250526100306'></a>
# [0.14.5-20250526100306 (2025-05-26)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.14.5-20250526100306)
### Resolved Bugs

- [BUG-929654](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-929654): - Pods restarting due to health check timeout - [PR 513](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/513)

---
<br>



<a name='0.14.4-20250515065938'></a>
# [0.14.4-20250515065938 (2025-05-15)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.14.4-20250515065938)
### New Functionality

- [US-683178](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-683178): Adding OTLP default settings tracing to Vector Store - [PR 502](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/502)

### Resolved Bugs

- [BUG-927458](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-927458): OTLP pr2 - [PR 506](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/506)

---
<br>



<a name='0.14.3-20250423164352'></a>
# [0.14.3-20250423164352 (2025-04-23)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.14.3-20250423164352)
### Resolved Bugs

- [BUG-922750](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-922750): Fixed isolation not found error - [PR 490](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/490)

---
<br>



<a name='0.14.2-20250417132456'></a>
# [0.14.2-20250417132456 (2025-04-17)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.14.2-20250417132456)
### Resolved Bugs

- [BUG-922256](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-922256): Fixed schema_info function - [PR 488](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/488)

---
<br>



<a name='0.14.1-20250416071545'></a>
# [0.14.1-20250416071545 (2025-04-16)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.14.1-20250416071545)
### Resolved Bugs

- [BUG-921713](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-921713): uncomment file text enpoint - [PR 486](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/486)

---
<br>



<a name='0.14.0-20250414115458'></a>
# [0.14.0-20250414115458 (2025-04-14)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.14.0-20250414115458)
### New Functionality

- [US-677369](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-677369): genai llm config - [PR 484](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/484)
- [US-669304](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-669304): Adopt SC 0.5.0 changes - [PR 482](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/482)
- [US-647581](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-647581): updating golang version and packages - [PR 475](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/475)
- [US-676120](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-676120): -add label for MRDR simulation exclusion - [PR 478](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/478)
- [US-669892](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-669892): - Return documentsCount on /v2/isolationID/collections - [PR 474](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/474)
- [US-671721](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-671721): db auto update - [PR 464](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/464)

### Resolved Bugs

- [BUG-920134](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-920134): - VS is not working on fedramp . Missed token - [PR 483](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/483)
- [BUG-917573](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-917573): duplicate key 014 - [PR 472](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/472)
- [BUG-918935](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-918935): - Update ps-restapi version - [PR 471](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/471)
- [BUG-918617](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-918617): Fixed db-tools - [PR 466](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/466)

---
<br>



<a name='0.13.1-20250328164049'></a>
# [0.13.1-20250328164049 (2025-03-28)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.13.1-20250328164049)
### Resolved Bugs

- [BUG-917573](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-917573): Fixed table-prefix duplicate key error - [PR 470](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/470)
- [BUG-918935](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-918935): - Update ps-restapi version - [PR 469](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/469)
- [BUG-918617](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-918617): Fixed db-tools - [PR 465](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/465)

---
<br>



<a name='0.13.0-20250324174015'></a>
# [0.13.0-20250324174015 (2025-03-26)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.13.0-20250324174015)
### New Functionality

- [US-664535](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-664535): - FEDRAMP Support - [PR 458](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/458)
- [US-668631](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-668631): VS ro mode - [PR 447](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/447)
- [US-664535](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-664535): - Fedramp support - [PR 456](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/456)
- [US-664535](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-664535): - Fedramp Support - [PR 454](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/454)
- [US-665601](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-665601): list chunks - [PR 452](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/452)
- [US-654788](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-654788): -  Vector Store supports Bedrock Embeddings model - [PR 451](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/451)
- [US-652252](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-652252): helm plugin change to helm-cli, opinionated plugin version update and go version change - [PR 445](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/445)
- [US-654788](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-654788): - Vector Store supports Bedrock Embeddings model - [PR 444](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/444)
- [US-651737](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-651737): - SAX new versio adoption for Fedramp step 2 - [PR 442](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/442)
- [US-668248](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-668248): Added embedding profiles - [PR 440](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/440)

### Resolved Bugs

- [BUG-917789](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-917789): - change expression for gcp - [PR 463](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/463)
- [BUG-911570](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-911570): - Images in Production must be pulled from release repo - [PR 462](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/462)
- [BUG-916452](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-916452): DB tables locked on heavy load - [PR 457](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/457)
- [BUG-914465](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-914465): Create missed collections tables - [PR 449](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/449)
- [BUG-877901](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-877901): Increased max collection name len to 255 - [PR 443](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/443)

### Addressed Issues

- [ISSUE-132008](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-132008): Added copy_schema function - [PR 453](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/453)

---
<br>



<a name='0.12.2-20250314182953'></a>
# [0.12.2-20250314182953 (2025-03-14)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.12.2-20250314182953)
### Resolved Bugs

- [BUG-916452](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-916452): DB tables locks on heavy load - [PR 455](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/455)

---
<br>



<a name='0.12.1-20250303170320'></a>
# [0.12.1-20250303170320 (2025-03-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.12.1-20250303170320)
### Resolved Bugs

- [BUG-914465](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-914465): Create missed collections tables - [PR 448](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/448)

---
<br>



<a name='0.11.1-20250303161004'></a>
# [0.11.1-20250303161004 (2025-03-04)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.11.1-20250303161004)
### Resolved Bugs

- [BUG-914465](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-914465): Create missed collections tables - [PR 446](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/446)

---
<br>



<a name='0.12.0-20250226090511'></a>
# [0.12.0-20250226090511 (2025-02-26)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.12.0-20250226090511)
### New Functionality

- [US-662064](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-662064): upgrade helm - [PR 441](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/441)
- [US-665357](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-665357): Uncommented missed tests - [PR 435](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/435)
- [US-665357](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-665357): embedding attributes - [PR 430](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/430)
- [US-666173](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-666173): - enable smart chunking based on condition - [PR 433](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/433)

### Resolved Bugs

- [BUG-906203](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-906203): restrict isolation deletion for mrdr when DeploymentMode is other than active - [PR 434](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/434)
- [BUG-912912](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-912912): Fixed document status typo - [PR 432](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/432)

### Addressed Issues

- [ISSUE-131427](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issues/ISSUE-131427): Merge Feature/EPIC-100239 to main - [PR 431](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/431)

---
<br>



<a name='0.11.0-20250211170132'></a>
# [0.11.0-20250211170132 (2025-02-11)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.11.0-20250211170132)
### New Functionality

- [US-665296](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-665296): -remove smart chunking and smart chunking role sce - [PR 428](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/428)

---
<br>



<a name='0.10.0-20250207134618'></a>
# [0.10.0-20250207134618 (2025-02-07)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.10.0-20250207134618)
### New Functionality

- [US-664770](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-664770): -change dependency on smart chunking to optional - [PR 427](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/427)

---
<br>



<a name='0.9.1-20250129172715'></a>
# [0.9.1-20250129172715 (2025-01-29)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.9.1-20250129172715)
### New Functionality

- [US-632582-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-632582-1): [PATCH] update vector store with mrdr compliant autopilot - [PR 424](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/424)
- [US-658809](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-658809): [PATCH] Added old RELEASE_NOTES records - [PR 420](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/420)

### Resolved Bugs

- [BUG-908186](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-908186): Do not fail vector_store.lookup_resources_metadata() if missed collection table - [PR 422](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/422)
- [BUG-903175](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-903175): - Change PDB value - [PR 421](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/421)
- [BUG-903317](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-903317): - Remove regcred from dbtools - [PR 418](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/418)

---
<br>



<a name='0.9.0-20241230105440'></a>
# [0.9.0-20241230105440 (2025-01-02)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.9.0-20241230105440)
### Resolved Bugs

- [BUG-903685](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-903685): - Smart Chunking not working with PDF file - [PR 417](https://git.pega.io/projects/PCLD/repos/gen-ai-vector-store/pull-requests/417)

---
<br>


<a name='0.9.0-20241227142653'></a>
# [0.9.0-20241227142653 (2025-01-02)](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local%2Fcom%2Fpega%2Fcloudservices%2Fgenai-vector-store%2FGenaiVectorStore%2F0.9.0-20241227142653)
### Functionality

- [BUG-844789](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-844789) Do not fail when isolation or collection does not exist
- [US-588092](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-588092) Updated threat model

---
<br>

<a name='0.8.1-20241213122552'></a>
# [0.8.1-20241213122552](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [US-622696](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622696) Removed DB schema version 1 support
- [BUG-899766](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-899766) Improved error/log messages for endpoint file/text when smart-chunking service throwing 500 error.
- [BUG-898351](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-898351) Custom attributes are not persisted when PUT via /file/text
- [US-649803](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-649803) Updated Terraform version to 1.8.0
- [BUG-897831](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-897831) Set attribute type to default (string) if not provided
- [BUG-897135](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-897135) Refactored update status logic
- [BUG-897300](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-897300) Validate isolation before querying
- [BUG-899495](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-899495) (MRDR support) Do not return error if isolation exists in DB
- [BUG-899878](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-899878) Decreased DB load
- [BUG-898985](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-898985) Do not add smart attributes by default
- [BUG-898351](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-898351) Custom attributes are not persisted when PUT via /file/text

---
<br>

<a name='0.8.0-20241112163828'></a>
# [0.8.0-20241112163828](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [US-629034](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-629034) Updated DBInstance to 5.20.2-20241023075931
- [US-644162-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-644162-1) Adding OWASP threat model
- [BUG-896779](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-896779) Fixed image pull secret for db-tools
- [US-649803](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-649803) Update SAX to support Fedramp
- [US-617004](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-617004) Adjust IsolationID parameter

---
<br>


<a name='0.7.10-20241125205620'></a>
# [0.7.10-20241125205620](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-897300](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-897300) Validate isolation before querying

---
<br>


<a name='0.7.9-20241112094241'></a>
# [0.7.9-20241112094241](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-896779](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-896779) Fixed image pull secret for db-tools

---
<br>

<a name='0.7.8-20241106145826'></a>
# [0.7.8-20241106145826](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [ISSUE-128856](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issue/ISSUE-128856) Increased SCE deployment timeout
- [BUG-895992](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-895992) Fixed isolation deletion
- [BUG-895319](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-895319) Always return minimal attr_id in vs1_get_or_create_v2_attribute() in case of duplicates
- [US-629038](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-629038) Update service-base version

---
<br>


<a name='0.7.7-20241030094419'></a>
# [0.7.7-20241030094419](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [US-643826](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-643826) Include GenAI Smart Chunking into GenAI Vector Store Product
- [US-645032](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-645032) Added raw text handling endpoint ../file/text
- [ISSUE-128439](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issue/ISSUE-128439) improved logging
- [US-641256](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/US-641256) Added ./document/delete-by-id endpoint
- [BUG-893296](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-893296) Validate isolationID before chunking
- [US-647100](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-647100) Disable Smart-Chunking by default in version 0.7.x
- [ISSUE-128648](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issue/ISSUE-128648) Merge changes from 0.6.13 to 0.7.7
- [BUG-892713](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-892713) Fixed SAX for GCP
- [BUG-894803](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-894803) Fix embedding rescheduling (workaround)
- [ISSUE-128756](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issue/ISSUE-128756) Avoid starting in parallel in integtests
- [BUG-894601](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-894601) Fix expression in TestControlPlane Product
- [BUG-871282](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-871282) Fixed not correct embedding statuses

---
<br>


<a name='0.7.6-20241010192123'></a>
# [0.7.6-20241010192123](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-873774](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-873774)
  Take into account chunk level attributes when querying documents (backward compatibility)

---
<br>


<a name='0.7.5-20241008132908'></a>
# [0.7.5-20241008132908](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-891691](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-891691) Service account already exists

---
<br>

<a name='0.7.4-20241003172637'></a>
# [0.7.4-20241003172637](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [US-631580](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-631580) Added attributes group endpoint
- [US-639903](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-639903) Implement /v1/.../documents/{documentID}/file endpoint (SYNC)
- [BUG-890838](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-890838) findDocument returns no data
- [BUG-891177](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-891177) Improved list documents performance

---
<br>

<a name='0.7.3-20240923140451'></a>
# [0.7.3-20240923140451](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [US-639853](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-639853) Validate Schema version before starting service
- [US-622694](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622694) [Rewrite SQL] OPS Metrics
- [US-612454](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-612454) [Rewrite SQL] Reindex
- [US-622687](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622687) [Rewrite SQL] Patch document
- [US-622684](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622684) [Rewrite SQL] Put document
- [US-622691](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622691) [Rewrite SQL] List attributes
- [US-622682](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622682) [Rewrite SQL] Isolations
- [US-622701](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622701) [Rewrite SQL] embedding_queue table
- [BUG-849955](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-849955) Return 409 on POST if isolation already exists

---
<br>

<a name='0.7.2-20240909183639'></a>
# [0.7.2-20240909183639](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [US-612473-3](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-612473-3) Added GCP support
- [US-637831](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-637831) Use Cloud SQL Go Connector
- [US-634995](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-634995) Restructured internal packages
- [ISSUE-126815](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issue/ISSUE-126815) Improved find chunks query
- [BUG-873935](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-873935) Workaround for
  deleting documents with special characters in name
- [BUG-879805](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-879805) Fixed disk size
  calculation
- [BUG-880851](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-880851) Reconstructed version
  0.7.x
- [US-622689-3](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622689-3) [Rewrite SQL] Query chunks
- [US-622690-4](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622690-4) [Rewrite SQL] Query documents
- [US-622685-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622685-1) [Rewrite SQL] Delete document
- [US-622683-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622683-1) [Rewrite SQL] List documents
- [US-622686](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622686) [Rewrite SQL] Get document
- [US-622684](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-622684) [Rewrite SQL] Put document


---
<br>

<a name='0.6.15-HFIX-20241031230815'></a>
# [0.6.15-HFIX-20241031230815](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-895319](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-895319) Always return minimal attr_id in vs1_get_or_create_v2_attribute() in case of duplicates
- [BUG-895474](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-895474) Sync missed attributes

---
<br>

<a name='0.6.14-HFIX-20241025162539'></a>
# [0.6.14-HFIX-20241025162539](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-894407](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-894407) Reimplemented fix for attribute duplications removal"

---
<br>

<a name='0.6.13-HFIX-20241023113503'></a>
# [0.6.13-HFIX-20241023113503](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-893420](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-893420) Synchronize missed items
- [BUG-893709](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-893709) Fixed attributes duplication

---
<br>

<a name='0.6.12-HFIX-20241015133250'></a>
# [0.6.12-HFIX-20241015133250](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [ISSUE-128439](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issue/ISSUE-128439) improved logging

---
<br>

<a name='0.6.11-HFIX-20241002085338'></a>
# [0.6.11-HFIX-20241002085338](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-889145](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-889145) Fixed attributes synchronization calculation

---
<br>

<a name='0.6.8-HFIX-20240902092240'></a>
# [0.6.8-HFIX-20240902092240](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-884921](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-884921) Fixed attributes synchronization

---
<br>

<a name='0.6.7-HFIX-20240808140500'></a>
# [0.6.7-HFIX-20240808140500](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [BUG-873935](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-873935) Workaround for
  deleting documents with special characters in name
- [BUG-879805](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/bugs/BUG-879805) Fixed disk size
  calculation

---
<br>

<a name='0.6.6-HFIX-20240805163148'></a>
# [0.6.6-HFIX-20240805163148](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- Reconstructed version 0.6.x (BUG-880851)

---
<br>

<a name='0.6.0-HFIX-20240624201459'></a>
# [0.6.0-HFIX-20240624201459](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [ISSUE-124091](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/issue/ISSUE-124091) Update SDEA
  Opinionated plugins to 45.0.0 (Java 17)
- [US-612452-3](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-612452-3) Added Reverse DB
  schema synchronization
- [BUG-879529](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-625554) Added DBTools
  conditionally deployed

---
<br>

<a name='0.5.9-HFIX-20240808135323'></a>
# [0.5.9-HFIX-20240808135323](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality

- [US-609716-1](https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-609716-1) Added '/metrics'
  endpoint to expose metrics
- Added integration with GenAI Gateway
  Service [https://agilestudio.pega.com/prweb/AgileStudio/app/agilestudio/story/US-614715]
- Added '/metrics' endpoint to expose default system metrics
- Added '/metrics' endpoint to kubernetes
- Changed '/metrics' endpoint to 8082
- Added metrics for incoming HTTP requests
- Refactored Infrastructure SCE to use pegasec TF provided instead of okta.
- Added background processing
- Added Integration test for service
- Added linter
- Refactored project structure

---
<br>

<a name='0.4.1-20240617132150'></a>
# [0.4.1-20240617132150](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality
- Added SAX support

---
<br>

<a name='0.3.0-20240213174330'></a>
# [0.3.0-20240213174330](https://bin.pega.io/ui/repos/tree/General/cloudservices-release-local/com/pega/cloudservices/genai-vector-store)
### Functionality
- Added Integration test for Ops Service
- Added ServiceAuthenticationClientService SCE
- Added GenAIVectorStoreInfrastructure SCE
- Added GenAIVectorStoreSaxRegistration SCE
- Added GenAIVectorStoreIsolation SCE
- Added Ops Service

---
<br>

