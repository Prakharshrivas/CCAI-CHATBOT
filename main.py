from json import JSONDecodeError
from fastapi import FastAPI, Request
import requests
from sample_database import getname, getname1
from dialogflow import detect_intent_texts, run_sample
from pydantic import BaseModel
from test_dtmf import extract_number

app = FastAPI()


sample_json = {
    "fulfillmentResponse": {
        "messages": [
            {
                "text": {
                    "text": ["{insert your message here}"]
                }
            }
        ]
    },
    "pageInfo": {
        "key1": "value1",
        "key2": "value2"
    },
    "sessionInfo": {
        "key1": "value1",
        "key2": "value2"
    }
}



@app.post("/webhook")
async def root(request: Request):
  payload_as_json = await request.json()
  print("====================/webhook====================")
  print(payload_as_json)

  getintent=payload_as_json['intentInfo']['displayName']
  text=payload_as_json['text']
  gettag = payload_as_json['fulfillmentInfo']['tag']
  #print(gettag)

  if gettag == "from-start":
    sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]= getname(gettag,text)

  elif payload_as_json['intentInfo']['displayName']=="for-repeat":
    sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]= "ok"

  else:
    sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]= getname(getintent,text)
  #print(sample_json)
  #print("hi from arya's mistake")
  sample_json['pageInfo']=payload_as_json['pageInfo']
  sample_json['sessionInfo']=payload_as_json['sessionInfo']
  print(sample_json)
  sample_json['sessionInfo']['parameters']['last_response']=sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]
  print(sample_json)
  return sample_json



@app.post("/DBN_speech")
async def root2(request:Request):
  print("====================/DBN_speech====================")
  payload_as_json= await request.json()

  if payload_as_json['intentInfo']['displayName']== "dbn_speech":

    if (list(payload_as_json['intentInfo']['parameters'].keys())[0]) =="last-name-entity":
      sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]="doe"

    elif (list(payload_as_json['intentInfo']['parameters'].keys())[0]) =="first-name-entity":
      sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]="john"

    elif (list(payload_as_json['intentInfo']['parameters'].keys())[0]) =="first-last-name-entity":
      sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]="john doe"

    elif (list(payload_as_json['intentInfo']['parameters'].keys())[0]) =="last-first-name-entity":
      sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]="doe john"

  sample_json['pageInfo']=payload_as_json['pageInfo']
  sample_json['sessionInfo']=payload_as_json['sessionInfo']
  sample_json['sessionInfo']['parameters']['last_response']=sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]
  print(sample_json)
    
  return sample_json
  
@app.post("/DTMF")
async def root3(request:Request):
  print("====================/DTMF====================")
  payload_as_json= await request.json()
  #print(payload_as_json)
  if payload_as_json['pageInfo']['displayName']=="order-check-dbn" or payload_as_json['pageInfo']['displayName']=="check with directory" or payload_as_json['pageInfo']['displayName']=="op" :
    conv_dtmf=getname1(payload_as_json['intentInfo']['displayName'])
    if "first few" in payload_as_json['text']:
      conv_dtmf=conv_dtmf[:3]
    sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]= conv_dtmf

  else:
    sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]=" "

  sample_json['pageInfo']=payload_as_json['pageInfo']
  sample_json['sessionInfo']=payload_as_json['sessionInfo']
  sample_json['sessionInfo']['parameters']['last_response']=sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]
  print(sample_json)
    
  return sample_json

@app.post("/directory")
async def root4(request:Request):
  print("====================/directory====================")
  payload_as_json= await request.json()
  print(payload_as_json)
  if "john doe" and "press" in payload_as_json["text"].lower() or "doe" and "press" in payload_as_json["text"].lower() or "john" and "press" in payload_as_json["text"].lower():
    ext_no = extract_number(payload_as_json["text"])
    sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]=ext_no
  else:
    sample_json['fulfillmentResponse']['messages'][0]['text']['text'][0]="yes"
  sample_json['pageInfo']=payload_as_json['pageInfo']
  sample_json['sessionInfo']=payload_as_json['sessionInfo']
  print(sample_json)
  return sample_json

