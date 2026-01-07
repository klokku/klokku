# Architecture

Klokku follows a clean architecture pattern with clear separation of concerns between different layers:

## Technology Stack
- **Backend**: Go (Golang)
- **Database**: Postgres with migrations managed by golang-migrate
- **Web Framework**: Gorilla Mux for HTTP routing
- **Frontend**: [React application](https://github.com/klokku/klokku-ui), optionally served by the Go backend
- **External Integrations**: 
  - Google Calendar API

## Application Layers
1. **Domain Layer**: Contains the core business entities and logic
    - Domain models (e.g., Budget, Event, User)
    - Business rules and validations

2. **Repository Layer**: Handles data access and persistence
    - Database operations
    - Data mapping between domain models and database schema
    - Transaction management

3. **Service Layer**: Implements business logic and orchestrates operations
    - Business workflows
    - Integration with external services
    - Authorization and validation

4. **API Layer**: Handles HTTP requests and responses
    - Request parsing and validation
    - Response formatting
    - Error handling
    - Authentication middleware

## Key Design Patterns
- **Dependency Injection**: Components receive their dependencies through constructors
- **Repository Pattern**: Data access is abstracted behind interfaces
- **DTO Pattern**: Data Transfer Objects separate API representation from domain models
- **Middleware**: HTTP request processing pipeline for cross-cutting concerns

## Project Structure
- `/pkg/`: Contains the core packages of the application
    - `/budget/`: Budget management functionality
    - `/event/`: Event tracking functionality
    - `/user/`: User management functionality
    - `/stats/`: Statistics generation functionality
    - `/google/`: Google Calendar integration
- `/internal/`: Internal packages not meant for external use
- `/migrations/`: Database migration scripts
- `/storage/`: Data storage location
- `/contributing/`: Contribution guidelines

## API Design
The application exposes a RESTful API for frontend communication, with endpoints for:
- User management
- Budget management
- Event tracking
- Statistics generation
- Google Calendar integration
