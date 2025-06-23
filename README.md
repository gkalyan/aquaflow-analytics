# AquaFlow Analytics

**Daily Operations Assistant for Water Districts**

AquaFlow Analytics helps water operations managers like Olivia answer operational questions in seconds instead of hours. The system provides a natural language interface to query real-time SCADA data and generate operational reports.

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Custom TimescaleDB image: `timescale-postgis-pg17`
- Git

### Development Setup

1. **Clone and start services:**
   ```bash
   git clone https://github.com/gkalyan/aquaflow-analytics.git
   cd aquaflow-analytics
   docker-compose up -d
   ```

2. **Access the application:**
   - Frontend UI: http://localhost:5173
   - Backend API: http://localhost:3000
   - PgAdmin: http://localhost:8080 (admin@aquaflow.com / admin)
   - Database: localhost:5432 (aquaflow / aquaflow_dev)

3. **Test the setup:**
   ```bash
   # Test API health
   curl http://localhost:3000/health
   
   # Check database schema
   curl http://localhost:3000/api/schema
   ```

### Database Schema

**Important**: The aquaflow database schema and all tables already exist and have been manually created. The migration files in `db/migrations/` are for reference only and are not automatically executed.

### Architecture

- **Frontend**: Vue.js 3 + Vite + Tailwind CSS
- **Backend**: Go + Gin framework
- **Database**: TimescaleDB with PostGIS extensions
- **Cache**: Redis
- **Deployment**: Docker Compose

## 📋 Project Status & Milestones

- **📊 [View Detailed Milestones](docs/MILESTONES.md)** - Complete project roadmap with phases, success criteria, and technical details
- **🚀 [Quick Progress Dashboard](docs/PROGRESS.md)** - Weekly updated status, completed features, and next steps

**Current Status**: Phase 1 Complete (Authentication & UI) → Phase 2 Planning (Core Query System)

### Key Features

- 🤖 Natural language query interface
- ⚡ Sub-2-second response times
- 📊 Real-time SCADA data integration
- 📈 Morning check and weekly report templates
- 🔄 30-second polling for live updates
- 📱 Mobile-responsive design

### Development Commands

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f backend

# Enter backend container
docker-compose exec backend sh

# Run tests
docker-compose exec backend go test ./...
docker-compose exec frontend npm test

# Rebuild after changes
docker-compose restart backend
```

### Project Structure

```
aquaflow-analytics/
├── backend/                 # Go API server
│   ├── cmd/api/            # Main application
│   ├── internal/           # Internal packages
│   └── go.mod              # Go dependencies
├── frontend/               # Vue.js application
│   ├── src/                # Source code
│   └── package.json        # Node dependencies
├── db/                     # Database files
│   └── migrations/         # SQL migration files
├── docker/                 # Docker configurations
├── .claude/               # Claude Code commands
└── docker-compose.yml     # Service orchestration
```

### Demo Data

The system includes realistic demo data for Turlock Irrigation District operations including:
- Main Canal flow rates (800-1200 CFS)
- Don Pedro Reservoir levels (750-810 feet)
- Pump station pressures (20-80 PSI)
- Weather and efficiency metrics

### Claude Code Commands

This project supports Claude Code for rapid development:

- `CC-ISSUE: [description]` - Create GitHub issue
- `CC-IMPLEMENT: #[number]` - Implement feature
- `CC-TEST` - Run tests
- `CC-SHIP` - Deploy feature
- `CC-DOCKER: up` - Start services

### Success Metrics

- Query response time: < 2 seconds
- Data freshness: < 60 seconds
- System uptime: > 99.5%
- User satisfaction: Reduce question response time from 30 minutes to 30 seconds

### Contributing

This project follows a ship-daily development approach:
1. Create issues for new features
2. Implement in feature branches
3. Test thoroughly
4. Ship working features daily

### License

Copyright 2025 AquaFlow Analytics. All rights reserved.