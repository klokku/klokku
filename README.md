![Klokku Logo](https://klokku.com/klokku-github-banner.png)

# Klokku - track your time and balance your life

Klokku is an application designed to help you achieve a balanced lifestyle by optimizing daily routines and tracking time usage.
You can easily create and adjust time budgets for task groups and enable a structured approach to planning.
You can update your plan weekly, which gives you flexibility and ensures the plan remains realistic and aligned with your life’s demands.

Klokku provides a tool to monitor time allocation, offering insights into how time is spent and helping users make informed adjustments for continuous improvement. 

Read more on [klokku.com](https://klokku.com).

## Running Klokku

### Nightly/Development version

The easiest way to run Klokku is using Docker Compose.

1. Install Docker (newer versions have Docker Compose built-in).
2. Clone this repository.
3. Navigate to the cloned directory.
4. Copy the `.env.template` file to `.env` and adjust the configuration as needed.
5. Open the console.
6. Run the following command in the directory where you have placed the files:
    ```shell
    docker compose up -d
    ```

You can now access Klokku at http://localhost:8181 🚀


### Production version

Klokku currently does not have a production version.\
The domain model and the API are still in development and may change.

You can run a development version of Klokku to check out the features.\
This version is fully usable, but I cannot guarantee the stability of the API, nor the automatic data migration if the underlying model changes.