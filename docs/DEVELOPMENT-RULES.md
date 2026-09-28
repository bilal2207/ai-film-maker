# AI Filmmaker - Development Rules & Principles

All contributors and automated agents must adhere to the following principles:

1. **Small Bounded Changes:** Make incremental, focused changes. Avoid broad refactoring unless explicitly scoped.
2. **No Architectural Changes Without Approval:** Maintain the locked technology stack and system topology.
3. **No New Frameworks Without Approval:** Evaluate whether standard libraries or existing tools suffice before adding third-party dependencies.
4. **Every Feature Requires Tests:** No feature is complete without automated unit/integration tests verifying happy and failure paths.
5. **No Secrets in Source Code:** All sensitive credentials, tokens, and keys must be injected via environment variables. Never commit `.env` files.
6. **PostgreSQL is the System of Record:** Relational metadata, project graphs, user entities, and asset references live in PostgreSQL.
7. **Binary Media Does Not Belong in PostgreSQL:** All video, audio, rendered frames, and raw assets must be stored in object storage (S3).
8. **S3 Will Be Used for Media Storage:** Scalable, durable object storage is the sole destination for media artifacts.
9. **AI Communicates with the Application Through Defined Contracts:** AI services must exchange strictly typed JSON or protobuf schemas.
10. **AI Must Not Directly Manipulate Frontend State:** AI generates structural intentions (Film DSL), which the frontend / editor core executes.
11. **Film DSL Will Become the Contract Between AI Decisions and Editing:** The Film DSL standardizes script, scene, shot, camera, cut, VFX, and audio definitions.
12. **Prefer Simple Architecture Until Scale Requires Complexity:** Avoid premature optimization, distributed queues, or complex clustering until actual metrics demand them.
