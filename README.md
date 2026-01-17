![Klokku Logo](https://klokku.com/klokku-github-banner.png)

# Klokku - track your time and balance your life

Klokku is an application designed to help you achieve a balanced lifestyle by optimizing daily routines and tracking time usage.
You can easily create and adjust time budgets for task groups and enable a structured approach to planning.
You can update your plan weekly, which gives you flexibility and ensures the plan remains realistic and aligned with your life’s demands.

Klokku provides a tool to monitor time allocation, offering insights into how time is spent and helping users make informed adjustments for continuous improvement. 

Read more on [klokku.com](https://klokku.com).

## Running Klokku

### Nightly/Development version

The easiest way to run Klokku is using Docker Compose.\
Newest versions of Docker Desktop include Docker Compose built-in.

You can use a script to run all in one command:

```bash
curl -sSL https://raw.githubusercontent.com/klokku/klokku/main/install.sh | bash
```

Or follow these steps:

1. Download [docker-compose.yml](https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/docker-compose.yml) to your local machine.
    ```
    curl -O https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/docker-compose.yml
    ```
2. Download database initialization file to `./db/init.sql`
    ```
    curl -sSL -o db/init.sql https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/db/init.sql
    ```
3. Download [.env.template](https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/.env.template) file to your local machine and rename it to `.env`
    ```
    curl -o .env https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/.env.template
    ```
4. Adjust the configuration in `.env` file to your needs.
5. Run the following command in the directory where you have placed the files:
    ```
    docker compose up -d
    ```

You can now access Klokku at http://localhost:8181 🚀


### Production version

Klokku currently does not have a production version.\
The domain model and the API are still in development and may change.

You can run a development version of Klokku to check out the features.\
The development version is fully usable, but we cannot guarantee the stability of the API, nor the automatic data migration if the underlying model changes.

## API Documentation

The API documentation is available via Swagger UI when the application is running:

- **Swagger UI**: http://localhost:8181/swagger/index.html
- **OpenAPI JSON**: http://localhost:8181/swagger/doc.json

To regenerate the Swagger documentation after making changes to the API:
```shell
make swagger
```