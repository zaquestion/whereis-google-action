package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Response struct {
	ResponseID  string `json:"responseId"`
	QueryResult struct {
		Action     string `json:"action"`
		QueryText  string `json:"queryText"`
		Parameters struct {
			Username string `json:"username"`
		} `json:"parameters"`
		AllRequiredParamsPresent bool   `json:"allRequiredParamsPresent"`
		FulfillmentText          string `json:"fulfillmentText"`
		FulfillmentMessages      []struct {
			Text struct {
				Text []string `json:"text"`
			} `json:"text"`
		} `json:"fulfillmentMessages"`
		OutputContexts []struct {
			Name       string `json:"name"`
			Parameters struct {
				UsernameOriginal string `json:"username.original"`
				Username         string `json:"username"`
			} `json:"parameters"`
		} `json:"outputContexts"`
		Intent struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"intent"`
		IntentDetectionConfidence float64 `json:"intentDetectionConfidence"`
		DiagnosticInfo            struct {
		} `json:"diagnosticInfo"`
		LanguageCode string `json:"languageCode"`
	} `json:"queryResult"`
	OriginalDetectIntentRequest struct {
		Source  string `json:"source"`
		Version string `json:"version"`
		Payload struct {
			IsInSandbox bool `json:"isInSandbox"`
			Surface     struct {
				Capabilities []struct {
					Name string `json:"name"`
				} `json:"capabilities"`
			} `json:"surface"`
			Inputs []struct {
				RawInputs []struct {
					Query     string `json:"query"`
					InputType string `json:"inputType"`
				} `json:"rawInputs"`
				Arguments []struct {
					RawText   string `json:"rawText"`
					TextValue string `json:"textValue"`
					Name      string `json:"name"`
				} `json:"arguments"`
				Intent string `json:"intent"`
			} `json:"inputs"`
			User struct {
				LastSeen    time.Time `json:"lastSeen"`
				Permissions []string  `json:"permissions"`
				Locale      string    `json:"locale"`
				UserID      string    `json:"userId"`
			} `json:"user"`
			Conversation struct {
				ConversationID    string `json:"conversationId"`
				Type              string `json:"type"`
				ConversationToken string `json:"conversationToken"`
			} `json:"conversation"`
			AvailableSurfaces []struct {
				Capabilities []struct {
					Name string `json:"name"`
				} `json:"capabilities"`
			} `json:"availableSurfaces"`
		} `json:"payload"`
	} `json:"originalDetectIntentRequest"`
	Session string `json:"session"`
}

type (
	contextOut struct {
		Name       string `json:"name"`
		Lifespan   int    `json:"lifespan"`
		Parameters struct {
		} `json:"parameters"`
	}

	inputValueData struct {
		Permissions []string `json:"permissions"`
		OptContext  string   `json:"optContext"`
		Type        string   `json:"@type"`
	}

	possibleIntent struct {
		Intent         string         `json:"intent"`
		InputValueData inputValueData `json:"inputValueData"`
	}

	initialPrompt struct {
		TextToSpeech string `json:"textToSpeech,omitempty"`
	}

	inputPrompt struct {
		InitialPrompts []initialPrompt `json:"initalPrompt"`
	}

	expectedInput struct {
		InputPrompt     inputPrompt      `json:"inputPrompt"`
		PossibleIntents []possibleIntent `json:"possibleIntents"`
	}

	data struct {
		Google google `json:"google"`
	}

	google struct {
		ExpectUserResponse bool            `json:"expectUserResponse"`
		ExpectedInputs     []expectedInput `json:"expectedInputs"`
	}

	simpleResponse struct {
		TextToSpeech string `json:"textToSpeech,omitempty"`
	}

	simpleResponses struct {
		SimpleResponses []simpleResponse `json:"simpleResponses,omitempty"`
	}

	fulfillmentMessage struct {
		Platform        string          `json:"platform,omitempty"`
		SimpleResponses simpleResponses `json:"simpleResponses,omitempty"`
	}

	Request struct {
		FulfillmentMessages []fulfillmentMessage `json:"fulfillmentMessages,omitempty"`
		FulfillmentText     string               `json:"fulfillmentText,omitempty"`
		Source              string               `json:"source,omitempty"`
		Data                data                 `json:"payload,omitempty"`
	}

	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
)

func (l Location) String() string {
	return fmt.Sprintf("%d,%d", l.Latitude, l.Longitude)
}

/*
{
  "conversationToken": "{\"state\":null,\"data\":{}}",
  "expectUserResponse": true,
  "expectedInputs": [
    {
      "inputPrompt": {
        "initialPrompts": [
          {
            "textToSpeech": "PLACEHOLDER_FOR_PERMISSION"
          }
        ],
        "noInputPrompts": []
      },
      "possibleIntents": [
        {
          "intent": "actions.intent.PERMISSION",
          "inputValueData": {
            "@type": "type.googleapis.com/google.actions.v2.PermissionValueSpec",
            "optContext": "To deliver your order",
            "permissions": [
              "NAME",
              "DEVICE_PRECISE_LOCATION"
            ]
          }
        }
      ]
    }
  ]
}

"possibleIntents": [
   {
     "intent": "actions.intent.PERMISSION",
     "inputValueData": {
       "@type": "type.googleapis.com/google.actions.v2.PermissionValueSpec",
       "optContext": "To deliver your order",
       "permissions": [
         "NAME",
         "DEVICE_PRECISE_LOCATION"
       ]
     }
   }
]
*/

func main() {
	http.HandleFunc("/dialogflow", func(w http.ResponseWriter, r *http.Request) {
		var resp Response
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(buf, &resp)
		if err != nil {
			log.Fatal(err)
		}
		defer r.Body.Close()
		log.Println(string(buf))
		headers := w.Header()
		headers.Set("Content-type", "application/json")
		switch resp.QueryResult.Action {
		case "input.permissions":
			if ok := checkPermission(&resp, "DEVICE_PRECISE_LOCATION"); !ok {
				requestLocationPermission(w, r)
			}
			encoder := json.NewEncoder(w)
			err := encoder.Encode(&Request{
				FulfillmentMessages: []fulfillmentMessage{
					{
						Platform: "ACTIONS_ON_GOOGLE",
						SimpleResponses: simpleResponses{
							[]simpleResponse{
								{TextToSpeech: "Who would you like to spy on?"},
							},
						},
					},
				},
			})
			if err != nil {
				log.Fatal(err)
			}
		case "input.distance":
			if ok := checkPermission(&resp, "DEVICE_PRECISE_LOCATION"); !ok {
				log.Println("bad permissions")
				return
			}
			time := distance(Location{}, Location{})
			respond(w, r, fmt.Sprintf("%s is %d away", resp.QueryResult.Parameters.Username, time))
		case "output.permission":
			checkPermission(&resp, "DEVICE_PRECISE_LOCATION")
			respond(w, r, "thank you, were good to go")
		}
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

/*
   "user": {
     "lastSeen": "2017-12-30T13:22:38Z",
     "permissions": ["DEVICE_PRECISE_LOCATION"],
     "locale": "en-US",
     "userId": "ABwppHExi4Hf9MXVT-xMhY358DDA37QWZZMr2WazJEbyS3vQkzhpLm52Q08KAiftr3nnz3wiacr_8MT4DkTDFqU67OyR"
   },

*/
func checkPermission(resp *Response, perm string) bool {
	for _, p := range resp.OriginalDetectIntentRequest.Payload.User.Permissions {
		if strings.EqualFold(perm, p) {
			log.Println("found perm", perm)
			return true
		}
	}
	return false
}

func requestLocationPermission(w http.ResponseWriter, r *http.Request) {
	log.Println("requesting location permissions")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(&Request{
		Source: "google",
		FulfillmentMessages: []fulfillmentMessage{
			{
				Platform: "ACTIONS_ON_GOOGLE",
				SimpleResponses: simpleResponses{
					[]simpleResponse{
						{TextToSpeech: "PLACEHOLDER_FOR_PERMISSION"},
					},
				},
			},
		},
		Data: data{
			Google: google{
				ExpectUserResponse: true,
				ExpectedInputs: []expectedInput{
					{
						InputPrompt: inputPrompt{
							InitialPrompts: []initialPrompt{
								{
									TextToSpeech: "PLACEHOLDER_FOR_PERMISSION",
								},
							},
						},
						PossibleIntents: []possibleIntent{
							{
								Intent: "actions.intent.PERMISSION",
								InputValueData: inputValueData{
									Type:        "type.googleapis.com/google.actions.v2.PermissionValueSpec",
									OptContext:  "To compare",
									Permissions: []string{"DEVICE_PRECISE_LOCATION"},
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

func getUserLocation(user string) Location {
	url := fmt.Sprintf("https://whereis.global/getLocation?user=%s", user)
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return Location{}
	}

	var loc Location
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(loc)
	if err != nil {
		log.Println(err)
		return Location{}
	}
	return loc
}

func distance(l1, l2 Location) int {
	fmt.Sprintf("https://maps.googleapis.com/maps/api/directions/json?origin=%s&destination=%s&key=%s", l1, l2, os.ExpandEnv("GOOGLE_MAPS_API_KEY"))
	return 5
}

func respond(w http.ResponseWriter, r *http.Request, speech string) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(&Request{
		FulfillmentMessages: []fulfillmentMessage{
			{
				Platform: "ACTIONS_ON_GOOGLE",
				SimpleResponses: simpleResponses{
					[]simpleResponse{
						{TextToSpeech: speech},
					},
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
