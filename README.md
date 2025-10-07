![Klokku Logo](https://klokku.com/klokku-github-banner.png)

# Klokku - track your time and balance your life

Klokku is an application designed to help you achieve a balanced lifestyle by optimizing daily routines and tracking time usage.
You can easily create and adjust time budgets for task groups and enable a structured approach to planning.
You can update your plan weekly, which gives you flexibility and ensures the plan remains realistic and aligned with your life’s demands.

Klokku provides a tool to monitor time allocation, offering insights into how time is spent and helping users make informed adjustments for continuous improvement. 

Read more on [klokku.com](https://klokku.com).

## How to install Klokku

1. Open the console.
2. Copy the following command:
    ```shell
    docker run -d --name klokku \
       -p 8181:8181 \
       -v storage:/app/storage \
       -e KLOKKU_HOST="http://localhost:8181" \
       ghcr.io/klokku/klokku:latest
    ```
3. Optionally, adjust your environmental variables (-e):
   - `KLOKKU_HOST`: the URL address of your Klokku application.
4. Run the command. 

You can now access Klokku at http://localhost:8181 🚀