# Phase 2 Specification: Core Query System

**Project**: AquaFlow Analytics  
**Phase**: 2 of 5  
**Target Timeline**: Week 3-4  
**Status**: Ready for Implementation  

## Overview

Phase 2 implements the core natural language query system that enables water operations managers to ask questions about their SCADA data in plain English and receive responses in under 2 seconds.

**Success Criteria**: Query response time <2 seconds, support for 10+ common queries, >95% accuracy for supported patterns.

---

## 🎯 Phase 2 Issues Breakdown

### Issue 1: Natural Language Query Parser Foundation
**Priority**: High  
**Estimated Effort**: 3-4 days  
**Dependencies**: None  

**Description**: Implement the core natural language query parsing system that converts user queries into structured database queries.

**Acceptance Criteria**:
- [ ] Parse basic time-based queries ("flow rate in the last hour")
- [ ] Handle location-based queries ("pump station 3 pressure")
- [ ] Support comparison queries ("current vs yesterday")
- [ ] Extract parameters: timeframe, location, metric type
- [ ] Return structured query object for backend processing
- [ ] Handle 10+ predefined query patterns
- [ ] Error handling for unsupported query types

**Technical Implementation**:
```go
// backend/internal/core/services/query_parser.go
type QueryParser struct {
    patterns []QueryPattern
}

type ParsedQuery struct {
    Type       QueryType
    Parameters map[string]interface{}
    TimeRange  TimeRange
    Location   string
    Metrics    []string
}
```

**Query Patterns to Support**:
1. "What is the current flow rate in Main Canal?"
2. "Show me pump station 3 pressure for the last 2 hours"
3. "How does today's reservoir level compare to yesterday?"
4. "Are there any alarms in the system?"
5. "What was the peak flow rate yesterday?"
6. "Show me the efficiency trend for this week"
7. "Is pump station 2 running normally?"
8. "What's the water temperature at intake?"
9. "Show me system status overview"
10. "Generate morning check report"

**Files to Create/Modify**:
- `backend/internal/core/services/query_parser.go`
- `backend/internal/core/models/query.go`
- `backend/internal/core/services/query_parser_test.go`

---

### Issue 2: SCADA Data Integration Engine
**Priority**: High  
**Estimated Effort**: 4-5 days  
**Dependencies**: Issue 1  

**Description**: Build the backend query engine that retrieves real-time SCADA data from TimescaleDB based on parsed queries.

**Acceptance Criteria**:
- [ ] Execute parsed queries against TimescaleDB
- [ ] Return results in under 2 seconds for 95% of queries
- [ ] Handle time-series aggregations (avg, min, max, latest)
- [ ] Support multiple data types (numeric, text, boolean)
- [ ] Implement query result caching with Redis
- [ ] Add comprehensive error handling
- [ ] Support concurrent queries
- [ ] Generate sample SCADA data for testing

**Technical Implementation**:
```go
// backend/internal/core/services/query_engine.go
type QueryEngine struct {
    db    *sql.DB
    cache *redis.Client
}

type QueryResult struct {
    Data      []map[string]interface{}
    Metadata  QueryMetadata
    Duration  time.Duration
    Cached    bool
}
```

**Database Integration**:
- Query `aquaflow.numeric_values` for real-time data
- Use TimescaleDB time_bucket for aggregations
- Implement connection pooling for performance
- Add query optimization for common patterns

**Sample Data Structure**:
```sql
-- Create realistic test data
INSERT INTO aquaflow.numeric_values (timestamp, series_id, value) VALUES
  ('2025-06-22 08:00:00', 1, 950.5),  -- Main Canal Flow Rate (CFS)
  ('2025-06-22 08:00:00', 2, 785.2),  -- Don Pedro Reservoir Level (ft)
  ('2025-06-22 08:00:00', 3, 45.8);   -- Pump Station 3 Pressure (PSI)
```

**Files to Create/Modify**:
- `backend/internal/core/services/query_engine.go`
- `backend/internal/core/services/cache_service.go`
- `backend/internal/core/handlers/query.go`
- `backend/internal/core/services/query_engine_test.go`
- `db/sample_data/scada_test_data.sql`

---

### Issue 3: Query API Endpoints Implementation
**Priority**: High  
**Estimated Effort**: 2-3 days  
**Dependencies**: Issue 1, Issue 2  

**Description**: Create REST API endpoints that handle natural language queries and return structured results.

**Acceptance Criteria**:
- [ ] POST `/api/query` endpoint for natural language queries
- [ ] GET `/api/query/templates` for predefined quick queries
- [ ] GET `/api/query/history` for user query history
- [ ] Response time consistently under 2 seconds
- [ ] Proper error handling and validation
- [ ] Request/response logging for debugging
- [ ] Rate limiting for query endpoints

**API Specification**:
```json
POST /api/query
{
  "query": "What is the current flow rate in Main Canal?",
  "user_id": "admin"
}

Response:
{
  "success": true,
  "result": {
    "answer": "The current flow rate in Main Canal is 950.5 CFS",
    "data": [{"timestamp": "2025-06-22T08:00:00Z", "value": 950.5, "unit": "CFS"}],
    "query_type": "current_value",
    "duration_ms": 150
  },
  "cached": false
}
```

**Quick Query Templates**:
1. Morning Check Report
2. System Status Overview  
3. Current Alarms
4. Flow Rate Summary
5. Pump Station Status
6. Reservoir Levels
7. Yesterday's Peak Usage
8. Efficiency Metrics
9. Temperature Readings
10. Pressure Monitoring

**Files to Create/Modify**:
- `backend/internal/core/handlers/query.go`
- `backend/internal/core/models/api_models.go`
- `backend/internal/core/middleware/rate_limiter.go`
- Update `backend/cmd/api/main.go` with new routes

---

### Issue 4: Enhanced Frontend Query Interface
**Priority**: Medium  
**Estimated Effort**: 3-4 days  
**Dependencies**: Issue 3  

**Description**: Upgrade the dashboard with an intelligent query interface featuring autocomplete, quick templates, and result visualization.

**Acceptance Criteria**:
- [ ] Natural language query input with autocomplete
- [ ] Quick query template buttons for common operations
- [ ] Real-time query result display with formatting
- [ ] Query history with favorites functionality
- [ ] Loading states and error handling
- [ ] Mobile-responsive query interface
- [ ] Query suggestions based on typing

**UI Components**:
```vue
<!-- QueryInterface.vue -->
<template>
  <div class="query-interface">
    <QueryInput 
      v-model="currentQuery"
      :suggestions="suggestions"
      @submit="executeQuery"
    />
    <QuickTemplates 
      :templates="quickTemplates"
      @select="executeTemplate"
    />
    <QueryResults 
      :result="lastResult"
      :loading="isLoading"
    />
    <QueryHistory 
      :history="queryHistory"
      @select="loadHistoryQuery"
    />
  </div>
</template>
```

**Quick Template Examples**:
- "Show me current system status"
- "What are today's flow rates?"
- "Any alarms or issues?"
- "Generate morning check report"
- "Compare this week to last week"

**Features**:
- Auto-suggestion based on typing
- Voice input support (future enhancement)
- Query result export to PDF/Excel
- Shareable query links
- Real-time updates for live data queries

**Files to Create/Modify**:
- `frontend/src/components/QueryInterface.vue`
- `frontend/src/components/QueryInput.vue`
- `frontend/src/components/QuickTemplates.vue`
- `frontend/src/components/QueryResults.vue`
- `frontend/src/components/QueryHistory.vue`
- `frontend/src/stores/query.js`
- `frontend/src/services/queryApi.js`

---

### Issue 5: Query Result Visualization System
**Priority**: Medium  
**Estimated Effort**: 3-4 days  
**Dependencies**: Issue 4  

**Description**: Implement intelligent visualization of query results including charts, tables, and status indicators.

**Acceptance Criteria**:
- [ ] Automatic chart type selection based on data type
- [ ] Real-time updating charts for live data
- [ ] Interactive data tables with sorting/filtering
- [ ] Status indicators for operational metrics
- [ ] Export functionality for charts and data
- [ ] Responsive visualization for mobile devices
- [ ] Performance optimization for large datasets

**Visualization Types**:
1. **Time Series Charts**: Flow rates, pressure trends, temperature
2. **Gauge Charts**: Current levels, efficiency percentages
3. **Status Cards**: System health, alarm states
4. **Data Tables**: Historical data, detailed readings
5. **Comparison Charts**: Current vs historical data
6. **Map Views**: Geographic distribution of sensors (future)

**Technical Implementation**:
```vue
<!-- QueryVisualization.vue -->
<template>
  <div class="visualization-container">
    <TimeSeriesChart 
      v-if="isTimeSeriesData"
      :data="chartData"
      :options="chartOptions"
    />
    <StatusCard 
      v-else-if="isStatusData"
      :status="statusData"
    />
    <DataTable 
      v-else
      :data="tableData"
      :columns="tableColumns"
    />
  </div>
</template>
```

**Chart Library**: PrimeVue Charts (Chart.js integration)

**Files to Create/Modify**:
- `frontend/src/components/QueryVisualization.vue`
- `frontend/src/components/charts/TimeSeriesChart.vue`
- `frontend/src/components/charts/GaugeChart.vue`
- `frontend/src/components/StatusCard.vue`
- `frontend/src/components/DataTable.vue`
- `frontend/src/utils/chartUtils.js`
- `frontend/src/utils/dataFormatters.js`

---

### Issue 6: Query Performance Optimization & Caching
**Priority**: Medium  
**Estimated Effort**: 2-3 days  
**Dependencies**: Issue 2, Issue 3  

**Description**: Implement comprehensive caching and optimization strategies to ensure sub-2-second query response times.

**Acceptance Criteria**:
- [ ] Redis caching for frequently accessed queries
- [ ] Database query optimization with proper indexes
- [ ] Connection pooling for database connections
- [ ] Query result compression for large datasets
- [ ] Cache invalidation strategy for real-time data
- [ ] Performance monitoring and metrics collection
- [ ] Load testing for concurrent queries

**Caching Strategy**:
```go
// Cache keys based on query signature
cacheKey := fmt.Sprintf("query:%s:%s", queryHash, timeWindow)

// Cache TTL based on data freshness requirements
ttl := map[string]time.Duration{
    "realtime": 30 * time.Second,
    "hourly":   5 * time.Minute,
    "daily":    30 * time.Minute,
}
```

**Database Optimizations**:
- Add indexes on `timestamp`, `series_id`, `parameter_id`
- Use TimescaleDB continuous aggregates
- Implement query result pagination
- Add database connection pooling

**Files to Create/Modify**:
- `backend/internal/core/services/cache_service.go`
- `backend/internal/core/db/optimization.sql`
- `backend/internal/core/middleware/performance.go`
- `backend/internal/core/services/metrics.go`

---

### Issue 7: Comprehensive Testing & Documentation
**Priority**: Low  
**Estimated Effort**: 2-3 days  
**Dependencies**: All previous issues  

**Description**: Implement comprehensive testing suite and documentation for the Phase 2 query system.

**Acceptance Criteria**:
- [ ] Unit tests for query parser (>90% coverage)
- [ ] Integration tests for query engine
- [ ] End-to-end tests for query API
- [ ] Frontend component tests
- [ ] Performance benchmarks for query response times
- [ ] API documentation with examples
- [ ] User guide for natural language queries

**Testing Strategy**:
```go
// backend/tests/query_parser_test.go
func TestQueryParser_ParseTimeBasedQuery(t *testing.T) {
    parser := NewQueryParser()
    result := parser.Parse("flow rate in the last hour")
    
    assert.Equal(t, QueryTypeTimeSeries, result.Type)
    assert.Equal(t, "1h", result.TimeRange.Duration)
    assert.Equal(t, "flow_rate", result.Metrics[0])
}
```

**Documentation Requirements**:
- API endpoint documentation
- Supported query patterns reference
- Performance benchmarks
- Troubleshooting guide
- Development setup instructions

**Files to Create/Modify**:
- `backend/tests/` (multiple test files)
- `frontend/tests/` (multiple test files)
- `docs/API.md`
- `docs/QUERY_PATTERNS.md`
- `docs/PERFORMANCE.md`

---

## 🚀 Implementation Timeline

### Week 3 (Days 1-5)
- **Day 1-2**: Issue 1 - Query Parser Foundation
- **Day 3-4**: Issue 2 - SCADA Data Integration (Part 1)
- **Day 5**: Issue 3 - Query API Endpoints (Part 1)

### Week 4 (Days 1-5)  
- **Day 1**: Issue 2 - SCADA Data Integration (Part 2)
- **Day 2**: Issue 3 - Query API Endpoints (Part 2)
- **Day 3-4**: Issue 4 - Enhanced Frontend Interface
- **Day 5**: Issue 5 - Visualization System (Part 1)

### Buffer Days
- Issue 5 - Visualization (Part 2)
- Issue 6 - Performance Optimization
- Issue 7 - Testing & Documentation

---

## 🔧 Technical Architecture

### Query Flow
1. **Frontend**: User types natural language query
2. **API**: Receives query, validates, logs request
3. **Parser**: Converts natural language to structured query
4. **Cache**: Check if result exists in Redis
5. **Engine**: Execute against TimescaleDB if not cached
6. **Response**: Return formatted result to frontend
7. **Visualization**: Display appropriate chart/table/status

### Technology Stack
- **Backend**: Go, Gin, TimescaleDB, Redis
- **Frontend**: Vue.js 3, PrimeVue, Chart.js
- **Testing**: Go testing, Vitest for frontend
- **Caching**: Redis with TTL-based invalidation
- **Database**: TimescaleDB with optimized indexes

---

## 📊 Success Metrics

### Performance Targets
- **Query Response Time**: <2 seconds (95th percentile)
- **Cache Hit Rate**: >70% for common queries
- **Concurrent Users**: Support 10+ simultaneous queries
- **Query Accuracy**: >95% for supported patterns

### Functional Requirements
- **Supported Queries**: 10+ predefined patterns
- **Data Freshness**: <60 seconds for real-time queries
- **Error Handling**: Graceful degradation for unsupported queries
- **Mobile Support**: Responsive design for tablet/phone

---

## 🎯 Definition of Done

Phase 2 is complete when:
- [ ] All 7 issues implemented and tested
- [ ] Query response time consistently <2 seconds
- [ ] 10+ query patterns working with >95% accuracy
- [ ] Frontend interface is intuitive and responsive
- [ ] Comprehensive test coverage (>80%)
- [ ] Performance benchmarks documented
- [ ] User can perform morning check via natural language
- [ ] System handles concurrent queries without degradation

---

*Last Updated: June 22, 2025*  
*Ready for GitHub Issue Creation*  
*Estimated Total Effort: 18-22 days*