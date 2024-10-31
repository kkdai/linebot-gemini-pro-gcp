// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

var geminiKey string

func main() {
	geminiKey = os.Getenv("GOOGLE_GEMINI_API_KEY")
	channelSecret := os.Getenv("ChannelSecret")
	bot, err := messaging_api.NewMessagingApiAPI(
		os.Getenv("ChannelAccessToken"),
	)
	if err != nil {
		log.Fatal(err)
		return
	}
	blob, err := messaging_api.NewMessagingApiBlobAPI(os.Getenv("ChannelAccessToken"))
	if err != nil {
		log.Fatal(err)
		return
	}
	if err != nil {
		log.Fatal(err)
	}

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		log.Println("/callback called...")

		cb, err := webhook.ParseRequest(channelSecret, req)
		if err != nil {
			log.Printf("Cannot parse request: %+v\n", err)
			if errors.Is(err, webhook.ErrInvalidSignature) {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}

		log.Println("Handling events...")
		for _, event := range cb.Events {
			log.Printf("/callback called%+v...\n", event)

			switch e := event.(type) {
			case webhook.MessageEvent:
				switch message := e.Message.(type) {
				case webhook.TextMessageContent:
					switch e.Source.(type) {
					case webhook.UserSource:
						log.Println("1 on 1 message")
						res, err := GeminiChat(message.Text)
						if err != nil {
							log.Println("Got GeminiChat err:", err)
							continue
						}
						ret := printResponse(res)
						if _, err = bot.ReplyMessage(
							&messaging_api.ReplyMessageRequest{
								ReplyToken: e.ReplyToken,
								Messages: []messaging_api.MessageInterface{
									messaging_api.TextMessage{
										Text: ret,
									},
								},
							},
						); err != nil {
							log.Print(err)
						} else {
							log.Println("Sent text reply.")
						}
						return
					case webhook.GroupSource:
						group := e.Source.(webhook.GroupSource).GroupId
						log.Println("Group ID=", group)
						for _, mention := range message.Mention.Mentionees {
							log.Println("mention data=", mention)
							switch mention.GetType() {
							case "user":
								botMention := mention.(*webhook.UserMentionee)
								log.Println("Mentioned user ID=", botMention.UserId, " isSelf=", botMention.IsSelf)

								if botMention.IsSelf {
									if _, err = bot.ReplyMessage(
										&messaging_api.ReplyMessageRequest{
											ReplyToken: e.ReplyToken,
											Messages: []messaging_api.MessageInterface{
												messaging_api.TextMessage{
													Text: "你好，我是 Gemini Chat Bot，請問有什麼可以幫助您的嗎？",
												},
											},
										},
									); err != nil {
										log.Print(err)
									} else {
										log.Println("Sent text reply.")
									}

									return
								}
							}
						}
						log.Println("Group ----- end")
					case webhook.RoomSource:
						room := e.Source.(webhook.RoomSource).RoomId
						log.Println("Room ID=", room)
						log.Println("Room ----- end")
					}
				case webhook.StickerMessageContent:
					replyMessage := fmt.Sprintf(
						"sticker id is %s, stickerResourceType is %s", message.StickerId, message.StickerResourceType)
					if _, err = bot.ReplyMessage(
						&messaging_api.ReplyMessageRequest{
							ReplyToken: e.ReplyToken,
							Messages: []messaging_api.MessageInterface{
								messaging_api.TextMessage{
									Text: replyMessage,
								},
							},
						}); err != nil {
						log.Print(err)
					} else {
						log.Println("Sent sticker reply.")
					}
				case webhook.ImageMessageContent:
					content, err := blob.GetMessageContent(message.Id)
					if err != nil {
						log.Println("Got GetMessageContent err:", err)
						return
					}
					defer content.Body.Close()
					if err != nil {
						log.Println("Got GetMessageContent err:", err)
					}
					data, err := io.ReadAll(content.Body)
					if err != nil {
						log.Fatal(err)
					}
					ret, err := GeminiImage(data)
					if err != nil {
						ret = "無法辨識圖片內容，請重新輸入:" + err.Error()
					}
					if _, err = bot.ReplyMessage(
						&messaging_api.ReplyMessageRequest{
							ReplyToken: e.ReplyToken,
							Messages: []messaging_api.MessageInterface{
								messaging_api.TextMessage{
									Text: ret,
								},
							},
						}); err != nil {
						log.Print(err)
					} else {
						log.Println("Sent sticker reply.")
					}

				default:
					log.Printf("Unsupported message content: %T\n", e.Message)
				}
			default:
				log.Printf("Unsupported message: %T\n", event)
			}
		}
	})

	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	fmt.Println("http://localhost:" + port + "/")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
