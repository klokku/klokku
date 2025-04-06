![Klokku Logo](https://klokku.com/klokku-github-banner.png)

# Klokku - track your time and balance your life

Klokku is an application designed to help you achieve a balanced lifestyle by optimizing daily routines and tracking time usage.
You can easily create and adjust time budgets for task groups and enable a structured approach to planning.
You can update your plan weekly, which gives you flexibility and ensures the plan remains realistic and aligned with your life’s demands.

Klokku provides a tool to monitor time allocation, offering insights into how time is spent and helping users make informed adjustments for continuous improvement. 

Read more on [klokku.com](https://klokku.com).

## How to install Klokku

Currently, the Klokku installation process requires the Google calendar integration.

Make sure you have Google Calendar API Client configured in Google Cloud Console. 

### Prerequisites

Configuring Google Calendar API Client.
1. Log in to the Google Cloud Console.
2. Click **Select a project**.
3. In the pop-up window, click **New project**.
4. Insert your Project Name and Location.  
5. Click the hamburger menu on the left-hand side and select **APIs & Services**.
6. From the navigation menu, select **Library**.
7. In the search field, type in *Google Calendar API*.
8. Select the *Google Calendar API* tile.
9.  Click **Enable**.
10. From the navigation menu, select **Credentials**.
11. Click **+ Create credentials** button from the top ribbon and select **OAuth client ID**. 

    >**Note:** To create an OAuth client ID, you must have your consent screen configured. To create the Google Auth Platform: 
    >1.  Click **Configure consent screen**.
    >2. Insert the following information:\
        - App name (you can insert any name).\
        - User support email.
    >3. Click **Next**.
    >4. Select **External** and click **Next**.
    >5. Insert contact information.
    >6. Select the *I agree to the Google API Services: User Data Policy* checkbox and click **Continue**.
    >7. Click **Create**.
12. From the drop-down menu, select the *Web application* type.
13. Copy the `http://<KLOKKU_HOST>:<KLOKKU_PORT>/api/integrations/google/auth/callback` address to the *Authorized redirect URIs* field and define the following variables:
    - `KLOKKU_HOST`: the host of your Klokku instance
    - `KLOKKU_PORT`: the port of your Klokku instance 
14. Click **Create**.

    The pop-up window with the OAuth client credentials is displayed. Record your Client ID and Client secret values, or click the **Download JSON** button, to save the credentials in a JSON file. Use these credentials during the Klokku installation. 


 ### Installation

1. Open the console.
2. Copy the following command:
    ```shell
    docker run -d --name klokku \
       -p 8181:8181 \
       -v storage:/app/storage \
       -e KLOKKU_HOST="http://localhost:8181" \
       -e KLOKKU_GOOGLE_CLIENT_ID="<GOOGLE_CLIENT_ID>" \
       -e KLOKKU_GOOGLE_CLIENT_SECRET="<GOOGLE_CLIENT_SECRET>" \
       ghcr.io/klokku/klokku:latest
    ```
3. Define your environmental variables (-e):
   - `KLOKKU_HOST`: the URL address of your Klokku application.
   - `KLOKKU_GOOGLE_CLIENT_ID`: your Google Cloud Client ID
   - `KLOKKU_GOOGLE_CLIENT_SECRET`: your Google Cloud client secret.  
4. Run the command. 
