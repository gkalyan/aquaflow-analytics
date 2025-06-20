# AquaFlow Claude Code Commands

## Project Context
You are building AquaFlow Analytics, a system that helps water district operations managers like Olivia answer questions about their water systems in seconds instead of hours. The system uses:
- Docker Compose for all services (TimescaleDB with PostGIS, Redis, Go backend, Vue frontend)
- Custom TimescaleDB image: `timescale-postgis-pg17`
- Patent-safe time-series database where series are identified by hash values
- 30-second polling for real-time data
- Query templates for common questions

## Available Commands

### CC-ISSUE: Create GitHub Issue
When I type "CC-ISSUE: [feature description]", you will:
1. Analyze the existing codebase structure
2. Check the database schema in db/migrations
3. Present a plan as questions:
   - How does this help Olivia save time?
   - What's the simplest implementation that ships today?
   - What existing code can we reuse?
   - What are the test scenarios?
4. Upon approval, create a GitHub issue using `gh issue create` with:
   - Title: Clear, action-oriented
   - User story: "As Olivia, I want to..."
   - Acceptance criteria
   - Technical steps
   - Add to project: `gh issue edit ISSUE --add-project "AquaFlow MVP"`

### CC-IMPLEMENT: Implement GitHub Issue
When I type "CC-IMPLEMENT: #[issue-number]", you will:
1. Read the issue: `gh issue view [issue-number]`
2. Create feature branch: `git checkout -b feature/issue-[number]-description`
3. Implement the solution following the issue requirements
4. Ensure Docker services are running: `docker-compose up -d`
5. Write tests if applicable
6. Test the implementation
7. Commit with message: `feat: [description] (closes #[issue-number])`

### CC-TEST: Test Current Feature
When I type "CC-TEST", you will:
1. Identify what feature we're working on
2. Write appropriate tests (Go tests for backend, Jest for frontend)
3. Run tests in Docker:
   - Backend: `docker-compose exec backend go test ./...`
   - Frontend: `docker-compose exec frontend npm test`
4. Show results and fix any failures

### CC-SHIP: Ship Current Feature
When I type "CC-SHIP", you will:
1. Ensure all tests pass
2. Commit all changes
3. Push to GitHub: `git push origin feature/current-branch`
4. Create PR: `gh pr create --title "feat: [description]" --body "Closes #[issue]"`
5. Merge if appropriate: `gh pr merge --squash`
6. Move issue to Done in project board

### CC-DB: Database Operations
When I type "CC-DB: [operation]", you will:
1. For "migrate": Run migrations in Docker
2. For "seed": Generate demo data
3. For "query": Execute queries to test data
4. For "schema": Show current schema

### CC-DOCKER: Docker Operations
When I type "CC-DOCKER: [operation]", you will:
1. For "up": Start all services with `docker-compose up -d`
2. For "logs": Show logs for specific service
3. For "rebuild": Rebuild containers after changes
4. For "status": Show status of all services

### CC-QUICK: Quick Win Features
When I type "CC-QUICK", suggest a feature that:
1. Can be shipped in 30 minutes
2. Provides immediate value to Olivia
3. Uses existing code/patterns
4. Has minimal dependencies

## Implementation Patterns

### Backend Endpoint Pattern
```go
// internal/core/handlers/[feature].go
func (h *Handler) [Feature](c *gin.Context) {
    // 1. Parse request
    // 2. Validate input
    // 3. Query database
    // 4. Return JSON response
}
```

### Frontend Component Pattern
```vue
<template>
  <!-- Simple, functional UI -->
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/services/api'

// Reactive state
// API calls
// 30-second polling if needed
</script>
```

### Database Query Pattern
Use TimescaleDB continuous aggregates when possible for performance.

## Daily Workflow
1. Morning: Check what to ship today
2. Create issue with CC-ISSUE
3. Implement with CC-IMPLEMENT
4. Test with CC-TEST
5. Ship with CC-SHIP
6. Repeat for next feature

## Environment Setup
- Custom TimescaleDB image with PostGIS: `timescale-postgis-pg17`
- Database: aquaflowdb (schema: aquaflow)
- Backend API: http://localhost:3000
- Frontend UI: http://localhost:5173
- PgAdmin: http://localhost:8080
- Redis: localhost:6379

## Success Metrics
- Query response time: < 2 seconds
- Ship 3+ features daily
- All tests passing
- Olivia can answer operational questions in 30 seconds