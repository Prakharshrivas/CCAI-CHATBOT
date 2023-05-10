import os
os.environ["GOOGLE_APPLICATION_CREDENTIALS"]="/home/gt43/Downloads/geniebot/new-odbj-f3953417d8e1.json"


import argparse
import uuid

from google.cloud.dialogflowcx_v3beta1.services.agents import AgentsClient
from google.cloud.dialogflowcx_v3beta1.services.sessions import SessionsClient
from google.cloud.dialogflowcx_v3beta1.types import session


def run_sample():
    project_id = "new-odbj"
    location_id = "global"
    agent_id = "myagent-test"
    agent = f"projects/{project_id}/locations/{location_id}/agents/{agent_id}"
    session_id = uuid.uuid4()
    texts = ["Hello"]
    language_code = "en-us"
    detect_intent_texts(agent, session_id, texts, language_code)


def detect_intent_texts(agent, session_id, texts, language_code):
  
    session_path = f"{agent}/sessions/{session_id}"
    print(f"Session path: {session_path}\n")
    client_options = None
    agent_components = AgentsClient.parse_agent_path(agent)
    location_id = agent_components["location"]
    if location_id != "global":
        api_endpoint = f"{location_id}-dialogflow.googleapis.com:443"
        print(f"API Endpoint: {api_endpoint}\n")
        client_options = {"api_endpoint": api_endpoint}
    session_client = SessionsClient(client_options=client_options)

    for text in texts:
        text_input = session.TextInput(text=text)
        query_input = session.QueryInput(text=text_input, language_code=language_code)
        request = session.DetectIntentRequest(
            session=session_path, query_input=query_input
        )
        response = session_client.detect_intent(request=request)

        print("=" * 20)
        print(f"Query text: {response.query_result.text}")
        response_messages = [
            " ".join(msg.text.text) for msg in response.query_result.response_messages
        ]
        print(f"Response text: {' '.join(response_messages)}\n")


if __name__ == "__main__":
    run_sample()
