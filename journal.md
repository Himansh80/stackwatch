# StackWatch — Journal

> Append-only log of every session. Newest at bottom.

---

## Session 1 — 2026-08-16 (planning)

### Goals
- Build full proof plan for replacing Datadog + 12 other tools
- Establish strict modularity rules
- Lock verification contract
- Set up project structure

### Done
- ✅ Researched all 12 tools via web search (capabilities documented in RESEARCH.md)
- ✅ Read existing plans (MASTER-BUILD-PLAN, ENTERPRISE-ROADMAP, IOS-PLATFORM-MISSING-ENDPOINTS, FEATURE-PLAN, IOS-PLATFORM-vs-DATADOG-COMPARISON, IOS-PLATFORM-ROADMAP-2026-2027)
- ✅ Created project folder `C:\Users\himan\HermesProjects\stackwatch\`
- ✅ Initialized git repo (local)
- ✅ Wrote PROJECT.md (single source of truth for the project)
- ✅ Wrote MASTER_BUILD_PLAN.md (13 tiers, 26-35 sessions, each tier sub-features)
- ✅ Wrote MODULARITY_RULES.md (strict rules from Datadog/Stripe/GitHub)
- ✅ Wrote VERIFICATION_CONTRACT.md (what "done" means, 5-step cycle)
- ✅ Wrote RESEARCH.md (feature inventory of all 12 tools)
- ✅ Wrote README.md
- ✅ Wrote .gitignore (forbids .bak files)
- ✅ Set up git config (name: Himan Shukla, email: himanshukhandelwal944@gmail.com)

### Decisions
- **Project name:** stackwatch (existing domain: stackwatch.smarthomelab.fun)
- **Path:** `C:\Users\himan\HermesProjects\stackwatch\`
- **Module size rule:** No file > 500 lines (Datadog-style)
- **Verify before claim:** Every feature must have live test output in same turn
- **GitHub:** Pending — to be set up before Tier 0 commit
- **First build:** Tier 0 — Foundation (git repo, base Go project, base React project, DB schema, auth, single dummy endpoint, CI)

### Next
- GitHub repo setup (ask user for repo name preference)
- Tier 0 implementation (foundation)
- User testing per verification contract

### Notes
- User explicitly said no repeating the plan
- All 5 user questions answered: full scope, both deployments, hybrid OSS, all OS + mobile, no budget
- User doesn't want me to ask "are you sure" or repeat the plan
- User wants strict modularity enforcement
- User wants iterative: build → test → fix → next
