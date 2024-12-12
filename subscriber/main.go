package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v3"
	"github.com/ziti-livekit-example/lib/openziti"
)

var (
	roomClient      *lksdk.RoomServiceClient
	livekitEndpoint string = "wss://livekit.ziti.example:7880"
	livekitKey      string = "GAkDU6thrPsKgxl"
	livekitSecret   string = "tZnZDeJGubl3tHJDPNvqfrQfmEkKcduQo0l23u9HY57"
)

func main() {
	logger.InitFromConfig(&logger.Config{Level: "debug"}, "ziti-livekit")
	lksdk.SetLogger(logger.GetLogger())

	for {
		run()
	}
}

func run() {
	err := openziti.InitCon("subscriber")
	if err != nil {
		log.Print(err)
		return
	}

	err = connectToLivekit()
	if err != nil {
		log.Print(err)
		return
	}

	// Create a room
	roomName := "testroom"
	config := &livekit.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    1222 * 60, // 10 minutes
		MaxParticipants: 20,
	}

	_, err = roomClient.CreateRoom(context.Background(), config)
	if err != nil {
		log.Print(err)
		return
	}
	log.Printf("Room %s created", roomName)

	// Create livekit access token
	canPublish := false
	canSubscribe := true
	canPublishData := true
	canUpdateOwnMetadata := false
	identy := "subscriber"

	grants := &auth.VideoGrant{
		RoomJoin:             true,
		Room:                 roomName,
		CanPublish:           &canPublish,
		CanSubscribe:         &canSubscribe,
		CanPublishData:       &canPublishData,
		CanUpdateOwnMetadata: &canUpdateOwnMetadata,
	}
	token, err := createLivekitAccessToken(identy, grants)
	if err != nil {
		log.Print(err)
		return
	}

	// Join room with token
	roomCB := &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataReceived: func(data []byte, rp lksdk.DataReceiveParams) {
				// process received data
				onDataReceived(data)
			},
			OnTrackSubscribed: onTrackSubscribed,
		},
		OnParticipantDisconnected: func(*lksdk.RemoteParticipant) {
			log.Print("publisher has left, waiting for him to come back...")
		},
	}
	room := lksdk.NewRoom(roomCB)

	// Join room
	err = room.JoinWithToken(livekitEndpoint, token, lksdk.WithICETransportPolicy(webrtc.ICETransportPolicyRelay))
	if err != nil {
		log.Print(err)
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	<-sigChan
	room.Disconnect()
}

// This will use zitified websocket connection to connect to livekit
// The ziti identity used will be the one thats setup in openziti package of this project
// The zitification happens in forked websocket library inside lib
func connectToLivekit() error {
	roomClient = lksdk.NewRoomServiceClient(
		livekitEndpoint,
		livekitKey,
		livekitSecret,
	)

	// To test that the host/keys are correct, do a test request
	_, err := roomClient.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
	if err != nil {
		log.Print(err)
		return err
	}

	log.Print("Connected to Livekit")
	return nil
}

func createLivekitAccessToken(identy string, grants *auth.VideoGrant) (string, error) {
	// Generate a livekit token
	at := auth.NewAccessToken(livekitKey, livekitSecret)

	// Grant permissions
	at.AddGrant(grants).
		SetIdentity(identy).
		SetValidFor(500 * time.Hour)

	// Convert to jwt
	token, err := at.ToJWT()
	if err != nil {
		log.Print(err)
		return "", err
	}
	return token, err
}

func onDataReceived(data []byte) {
	log.Printf("Received data channel data: %s", string(data))
}

func onTrackSubscribed(track *webrtc.TrackRemote, publication *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
	tracks := fmt.Sprintf("%s-%s", rp.Identity(), track.ID())
	fmt.Println("new track ", tracks)
}
