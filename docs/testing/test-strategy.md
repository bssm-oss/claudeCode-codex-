# Test strategy

This repository uses fast unit and functional tests for configuration, auth persistence, workspace boundaries, git wrapper behavior, provider request construction, and device-auth login against mock HTTP servers. Real networked OpenAI and ChatGPT logins remain manual QA concerns, but CI still exercises the local functional path with mocks.
