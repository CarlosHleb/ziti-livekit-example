package main

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/logger"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/mediadevices/pkg/codec/openh264"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/sirupsen/logrus"
	"github.com/ziti-livekit-example/lib/openziti"
)

var (
	roomClient        *lksdk.RoomServiceClient
	room              *lksdk.Room
	livekitEndpoint   string = "wss://livekit.ziti.example:7880"
	livekitKey        string = "GAkDU6thrPsKgxl"
	livekitSecret     string = "tZnZDeJGubl3tHJDPNvqfrQfmEkKcduQo0l23u9HY57"
	width             int    = 1640
	height            int    = 900
	framerate         int    = 30
	redTriangle       *image.RGBA
	redTriangleBounds image.Rectangle
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	logger.InitFromConfig(&logger.Config{Level: "debug"}, "ziti-livekit")
	lksdk.SetLogger(logger.GetLogger())
	logrus.StandardLogger().Level = logrus.DebugLevel
	rand.Seed(time.Now().UnixNano())

	for {
		run()
		time.Sleep(1 * time.Second)
	}
}

func run() {
	err := openziti.InitCon("publisher")
	if err != nil {
		log.Print(err)
		return
	}

	err = connectToLivekit()
	if err != nil {
		log.Print(err)
		return
	}

	redTriangle = drawRedTriangle(width, height)
	redTriangleBounds = getRedTriangleBounds(*redTriangle)

	// Create a room
	roomName := "testroom"

	// Create livekit access token
	canPublish := true
	canSubscribe := false
	canPublishData := true
	canUpdateOwnMetadata := true
	identy := "publisher"

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
		},
		OnParticipantDisconnected: func(*lksdk.RemoteParticipant) {
			log.Print("subscriber has left, waiting for him to come back...")
		},
	}
	room = lksdk.NewRoom(roomCB)

	// Join room
	err = room.JoinWithToken(livekitEndpoint, token, lksdk.WithICETransportPolicy(webrtc.ICETransportPolicyRelay))
	if err != nil {
		log.Print(err)
		return
	}

	err = setMetadata()
	if err != nil {
		log.Print(err)
		return
	}

	err = publishTrack()
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

func setMetadata() error {
	n := 43
	data := make(map[string]int)
	data["randommetadata"] = n

	// Marshall
	payload, err := json.Marshal(data)
	if err != nil {
		log.Print(err)
		return err
	}

	room.LocalParticipant.SetMetadata(string(payload))
	return nil
}

func drawRedTriangle(width int, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill the background with white color
	col := color.RGBA{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255))}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, col)
		}
	}

	// Define the dimensions of the triangle
	triangleHeight := height / 3 // Triangle height as a fraction of the image height
	triangleBase := width / 2    // Triangle base as a fraction of the image width

	// Calculate the triangle's coordinates to center it
	triangleBaseY := height/2 + triangleHeight/2 // Bottom y-coordinate of the triangle
	triangleTipY := height/2 - triangleHeight/2  // Top y-coordinate of the triangle
	triangleBaseX1 := width/2 - triangleBase/2   // Left x-coordinate of the triangle base
	triangleBaseX2 := width/2 + triangleBase/2   // Right x-coordinate of the triangle base
	triangleTipX := width / 2                    // x-coordinate of the triangle tip

	// Define the red color for the triangle
	red := color.RGBA{255, 0, 0, 255}

	// Fill the triangle using a simple scanline algorithm
	for y := triangleTipY; y <= triangleBaseY; y++ {
		// Interpolate the x-coordinates of the left and right edges
		t := float64(y-triangleTipY) / float64(triangleBaseY-triangleTipY)
		leftX := int(float64(triangleTipX)*(1-t) + float64(triangleBaseX1)*t)
		rightX := int(float64(triangleTipX)*(1-t) + float64(triangleBaseX2)*t)

		// Draw horizontal line for the current scanline
		for x := leftX; x <= rightX; x++ {
			img.Set(x, y, red)
		}
	}
	return img
}

func getRedTriangleBounds(img image.RGBA) image.Rectangle {
	return img.Bounds()
}

// From this reader xh264 takes decorated screen captures
type ScreenCaptureReader struct{}

// Read() Returns a decorated image of the screenToCapture
func (c ScreenCaptureReader) Read() (image.Image, func(), error) {
	redTriangle = drawRedTriangle(width, height)
	return redTriangle, func() {}, nil
}

func trackOnBind(track *lksdk.LocalTrack) error {
	// Create h264 params
	params, err := openh264.NewParams()
	if err != nil {
		log.Print(err)
		return err
	}

	// Configure params
	params.BitRate = 2000000
	params.EnableFrameSkip = false
	params.UsageType = openh264.ScreenContentRealTime

	// Set Media properties
	buf := ScreenCaptureReader{}
	bounds := redTriangleBounds
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y
	prop := prop.Media{
		Video: prop.Video{
			Width:     width,
			Height:    height,
			FrameRate: float32(framerate),
		},
	}

	// build encoder
	enc, err := params.BuildVideoEncoder(buf, prop)
	if err != nil {
		log.Print(err)
		return err
	}
	ticker := time.NewTicker(time.Second / time.Duration(framerate))

	// Start the ticker
	for range ticker.C {
		select {
		default:
			// Get h264 encoded frame
			b, _, err := enc.Read()
			if err != nil {
				log.Print(err)
				return err
			}

			log.Print(len(b))
			// Send the frame trough track
			duration := time.Second / time.Duration(framerate)
			err = track.WriteSample(media.Sample{Data: b, Duration: duration}, &lksdk.SampleWriteOptions{})
			if err != nil {
				log.Print(err)
				return err
			}
		}

	}
	return nil
}

func publishTrack() error {
	// Create a local track
	track, err := lksdk.NewLocalTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})
	if err != nil {
		log.Print(err)
		return err
	}

	// On local track bind handler
	track.OnBind(func() {
		err := trackOnBind(track)
		if err != nil {
			log.Print("Failed to bind track: ", err)
		}
	})

	// Options for local track publish
	bwidth := redTriangleBounds.Max.X - redTriangleBounds.Min.X
	bheight := redTriangleBounds.Max.Y - redTriangleBounds.Min.Y
	options := &lksdk.TrackPublicationOptions{
		VideoWidth:  bwidth,
		VideoHeight: bheight,
	}

	// Publish local track
	_, err = room.LocalParticipant.PublishTrack(track, options)
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}
