# Advanced Userprefs Example

This example demonstrates more complex usage of the `userprefs` library, including:
- Custom JSON types for preferences.
- Usage of both PostgreSQL for storage and Redis for caching.
- Defining multiple preferences.
- Setting and getting preferences for multiple simulated users.
- Basic concurrent access demonstration.

## Prerequisites

1.  **Go**: Ensure Go (version 1.24 or later) is installed.
2.  **Docker and Docker Compose**: Required for running the PostgreSQL database.
3.  **Redis (Optional but Recommended for full example)**: The `docker-compose.yml` file now also includes a Redis service. When you run `docker-compose up -d`, Redis will be started alongside PostgreSQL. The example will automatically connect to it for caching.

## Running the Example

1.  **Start PostgreSQL using Docker Compose**:
    Navigate to this directory (`examples/advanced/`) in your terminal and run:
    ```bash
    docker-compose up -d
    ```
    This will start PostgreSQL and Redis containers, configured for this example.

2.  **Run the Go Program**:
    From this directory, execute:
    ```bash
    go run main.go
    ```

3.  **View Output**:
    The program will output logs showing preference definitions, settings for different users, and results of `GetAll` operations.

4.  **Stop Services (Optional)**:
    When you're done, you can stop and remove the PostgreSQL and Redis containers:
    ```bash
    docker-compose down
    ```
