package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/livekit/server-sdk-go/v2/pkg/samplebuilder"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
	"github.com/ziti-livekit-example/lib/openziti"
)

var (
	roomClient      *lksdk.RoomServiceClient
	livekitEndpoint string = "wss://livekit.ziti.example:7880"
	livekitKey      string = "GAkDU6thrPsKgxl"
	livekitSecret   string = "tZnZDeJGubl3tHJDPNvqfrQfmEkKcduQo0l23u9HY57"
)

func main() {
	// logger.InitFromConfig(&logger.Config{Level: "debug"}, "ziti-livekit")
	// lksdk.SetLogger(logger.GetLogger())
	// logrus.StandardLogger().Level = logrus.DebugLevel

	for {
		run()
		time.Sleep(1 * time.Second)
	}
}

func run() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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
	log.Print("Join successfull.")

	log.Print("conn state ", room.ConnectionState())
	log.Printf("remote participants %+v", room.GetRemoteParticipants())

	for _, p := range room.GetRemoteParticipants() {
		log.Print("identity of participant: ", p.Identity())

		if p.Identity() == "publisher" {
			log.Print("publisher tracks: ", p.TrackPublications())
		}
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
	log.Print("new track ", tracks)
	NewTrackWriter(track, rp.WritePLI, tracks)
}

const (
	maxVideoLate = 1000 // nearly 2s for fhd video
	maxAudioLate = 200  // 4s for audio
)

type TrackWriter struct {
	sb     *samplebuilder.SampleBuilder
	writer media.Writer
	track  *webrtc.TrackRemote
}

func NewTrackWriter(track *webrtc.TrackRemote, pliWriter lksdk.PLIWriter, fileName string) (*TrackWriter, error) {
	var (
		sb     *samplebuilder.SampleBuilder
		writer media.Writer
		err    error
	)
	log.Print("-------", track.Codec().MimeType)
	switch {
	case strings.EqualFold(track.Codec().MimeType, "video/vp9"):
		log.Print("vp999999999")
		sb = samplebuilder.New(maxVideoLate, &codecs.VP9Packet{}, track.Codec().ClockRate, samplebuilder.WithPacketDroppedHandler(func() {
			pliWriter(track.SSRC())
		}))
		// ivfwriter use frame count as PTS, that might cause video played in a incorrect framerate(fast or slow)
		writer, err = ivfwriter.New(fileName + ".ivf")

	case strings.EqualFold(track.Codec().MimeType, "video/h264"):
		sb = samplebuilder.New(maxVideoLate, &codecs.H264Packet{}, track.Codec().ClockRate, samplebuilder.WithPacketDroppedHandler(func() {
			pliWriter(track.SSRC())
		}))
		writer, err = h264writer.New(fileName + ".h264")

	case strings.EqualFold(track.Codec().MimeType, "audio/opus"):
		sb = samplebuilder.New(maxAudioLate, &codecs.OpusPacket{}, track.Codec().ClockRate)
		writer, err = oggwriter.New(fileName+".ogg", 48000, track.Codec().Channels)

	default:
		return nil, errors.New("unsupported codec type")
	}

	if err != nil {
		return nil, err
	}

	t := &TrackWriter{
		sb:     sb,
		writer: writer,
		track:  track,
	}
	go t.start()
	return t, nil
}

func (t *TrackWriter) start() {
	defer t.writer.Close()
	for {
		pkt, _, err := t.track.ReadRTP()
		if err != nil {
			break
		}
		t.sb.Push(pkt)

		for _, p := range t.sb.PopPackets() {
			// t.writer.WriteRTP(p)
			log.Print(len(p.Payload))
		}
	}
}
