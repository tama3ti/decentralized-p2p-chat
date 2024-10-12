package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/piheta/sept/backend/models"
	"github.com/pion/webrtc/v4"
)

var chosenIP string
var peerConnection *webrtc.PeerConnection
var ws *websocket.Conn
var Ips []string

// p1p2, connects to the signaling server
// p1 creates and sends offer to the chosen peer
// p2 creates and sends answer to p1
// p1 sends ICE candidates to p2
// p2 replies with his ICE candidates
// datachannel is made
func SnConnectionHandler() {
	initializePeerConnection()
	createDataChannel()

	var wg sync.WaitGroup
	wg.Add(1)
	go connectToSignalingServer(&wg)
	wg.Wait()
}

func handleError(err error) {
	fmt.Println("Unexpected error. Check console.")
	log.Println(err)
}

func initializePeerConnection() {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	var err error
	peerConnection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		handleError(err)
	}

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		sendICECandidate(candidate)
	})

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", state.String())
	})
}

func createDataChannel() {
	sendChannel, err := peerConnection.CreateDataChannel("foo", nil)
	if err != nil {
		handleError(err)
		return
	}
	fmt.Println("Data channel created:", sendChannel.Label())

	sendChannel.OnClose(func() {
		fmt.Println("sendChannel has closed")
	})

	sendChannel.OnOpen(func() {
		fmt.Println("sendChannel has opened")
		candidatePair, err := peerConnection.SCTP().Transport().ICETransport().GetSelectedCandidatePair()
		if err != nil {
			fmt.Println("Error getting candidate pair:", err)
		} else {
			fmt.Println("Selected candidate pair:", candidatePair)
		}
	})

	sendChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("%s: %s\n", sendChannel.Label(), string(msg.Data)) //* HANDLES RECIEVED P2P MESSAGE
	})

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			MessagingHandler(d)
		})
	})

	// Add state change handler for the peer connection
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())
	})

	// Add state change handler for the ICE connection
	peerConnection.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", s.String())
	})
}

func connectToSignalingServer(wg *sync.WaitGroup) {
	defer wg.Done()

	var err error
	ws, _, err = websocket.DefaultDialer.Dial("ws://127.0.0.1:8081/ws", nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer ws.Close()

	//! Get and send cert to sig
	announceMessage, err := createAnnounceRequest()
	if err != nil {
		log.Fatalf("Failed to create announce request: %v", err)
	}

	err = ws.WriteMessage(websocket.TextMessage, announceMessage)
	if err != nil {
		log.Fatalf("Failed to send user data: %v", err)
	}

	//* Listen for messages from sig
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		var sigMessage models.SigMsg
		if err := json.Unmarshal(message, &sigMessage); err != nil {
			log.Printf("Failed to unmarshal sigMessage: %v", err)
			continue
		}

		switch sigMessage.Type {
		case models.UserSearch:
			// todo check jwt signature, add user to db
			fmt.Println(sigMessage.Data)
		case models.Connection:

			dataBytes, err := json.Marshal(sigMessage.Data)
			if err != nil {
				log.Printf("Failed to marshal Data: %v", err)
				return
			}

			var connectionRequest models.ConnectionRequest
			if err := json.Unmarshal(dataBytes, &connectionRequest); err != nil {
				log.Printf("Failed to unmarshal AnnounceRequest: %v", err)
				return
			}

			// todo, send these as sigmsg
			switch connectionRequest.Type {
			case "offer":
				onOffer(connectionRequest)
			case "answer":
				onAnswer(connectionRequest)
			case "candidate":
				onCandidate(connectionRequest)
			}

		default:
			fmt.Println("Unknown message type:", sigMessage.Type)
		}
	}
}

func SearchAndOffer(username string) (models.User, error) {
	req := models.SigMsg{
		Type: models.UserSearch,
		Data: models.UserSearchRequest{
			Username: username,
		},
	}

	jsonReq, err := json.Marshal(req)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to search request: %v", err)
	}

	err = ws.WriteMessage(websocket.TextMessage, jsonReq)
	if err != nil {
		log.Fatalf("Failed to send user data: %v", err)
	}

	// sendOffer()

	return models.User{}, nil
}

func createAnnounceRequest() ([]byte, error) {
	cert, err := os.ReadFile("./sept_data/user.jwt")
	if err != nil {
		return nil, fmt.Errorf("failed to get user cert: %v", err)
	}

	req := models.SigMsg{
		Type: models.Announce,
		Data: models.AnnounceRequest{
			Cert: string(cert),
		},
	}

	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %v", err)
	}

	return jsonReq, nil
}

//
// ICE HANDLING
//

// ICE
// Senders
func sendOffer() {
	cr := models.ConnectionRequest{
		Type:   "offer",
		DestIP: chosenIP,
		Data:   createOffer(),
	}
	crBytes, err := json.Marshal(cr)
	if err != nil {
		log.Fatalf("Failed to marshal ConnectionRequest: %v", err)
	}

	err = ws.WriteMessage(websocket.TextMessage, crBytes)
	if err != nil {
		log.Fatalf("Failed to send chosen IP: %v", err)
	}
}

func sendAnswer(destIP, answer string) {
	cr := models.ConnectionRequest{
		Type:   "answer",
		DestIP: destIP,
		Data:   answer,
	}
	crBytes, err := json.Marshal(cr)
	if err != nil {
		log.Fatalf("Failed to marshal ConnectionRequest: %v", err)
	}

	err = ws.WriteMessage(websocket.TextMessage, crBytes)
	if err != nil {
		log.Fatalf("Failed to send answer: %v", err)
	}
}

func sendICECandidate(candidate *webrtc.ICECandidate) {
	candidateJSON, err := json.Marshal(candidate.ToJSON())
	if err != nil {
		handleError(err)
		return
	}

	cr := models.ConnectionRequest{
		Type:      "candidate",
		Candidate: candidate,
		DestIP:    chosenIP,
		Data:      string(candidateJSON),
	}
	crBytes, err := json.Marshal(cr)
	if err != nil {
		handleError(err)
		return
	}

	if err := ws.WriteMessage(websocket.TextMessage, crBytes); err != nil {
		handleError(err)
	}
}

// ICE
// Recievers

func onOffer(connectionRequest models.ConnectionRequest) {
	fmt.Println("Received offer:", connectionRequest)
	answer := createAnswer(connectionRequest.Data)
	sendAnswer(*connectionRequest.SrcIP, answer)
}

func onAnswer(connectionRequest models.ConnectionRequest) {
	fmt.Println("Received answer:", connectionRequest.Data)
	answerBytes, err := base64.StdEncoding.DecodeString(connectionRequest.Data)
	if err != nil {
		handleError(err)
	}
	answerSDP := string(answerBytes)
	answerDesc := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answerSDP,
	}
	if err := peerConnection.SetRemoteDescription(answerDesc); err != nil {
		handleError(err)
	}
}

func onCandidate(connectionRequest models.ConnectionRequest) {
	fmt.Println("Received ICE candidate")
	candidate := webrtc.ICECandidateInit{}
	if err := json.Unmarshal([]byte(connectionRequest.Data), &candidate); err != nil {
		handleError(err)
		return
	}
	chosenIP = *connectionRequest.SrcIP // Replace "none" with the sender of the offer
	if err := peerConnection.AddICECandidate(candidate); err != nil {
		handleError(err)
	}
}

// ICE
// Helpers

func createOffer() string {
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		handleError(err)
	}
	if err := peerConnection.SetLocalDescription(offer); err != nil {
		handleError(err)
	}

	return base64.StdEncoding.EncodeToString([]byte(offer.SDP))
}

func createAnswer(offerBase64 string) string {
	offerBytes, err := base64.StdEncoding.DecodeString(offerBase64)
	if err != nil {
		handleError(err)
	}
	offerSDP := string(offerBytes)

	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerSDP,
	}
	if err := peerConnection.SetRemoteDescription(offer); err != nil {
		handleError(err)
	}

	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		handleError(err)
	}

	if err := peerConnection.SetLocalDescription(answer); err != nil {
		handleError(err)
	}

	return base64.StdEncoding.EncodeToString([]byte(answer.SDP))
}
