package main

import (
	"encoding/binary"
	log "github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"github.com/urfave/cli"
	"io"
	"math/rand"
	"os"
	"time"
)

const SESSIONS = 3

func action(cc *cli.Context) error {
	token := cc.String("token")
	if token == "" {
		log.Fatal("No token provided!")
	}

	gid := cc.String("guild")
	if gid == "" {
		log.Fatal("No guild provided!")
	}

	cid := cc.String("channel")
	if cid == "" {
		log.Fatal("No channel provided!")
	}

	log.WithFields(log.Fields{"cid": cid, "gid": gid}).Info("Starting...")

	f, err := os.Open("clip.dca")
	if err != nil {
		log.WithError(err).Fatal("Couldn't open audio file")
	}

	var frames [][]byte
	var frameSize int16
	for {
		if err := binary.Read(f, binary.LittleEndian, &frameSize); err != nil {
			if err == io.EOF {
				break
			}
			log.WithError(err).Fatal("Couldn't read frame size")
		}

		frame := make([]byte, frameSize)
		if err := binary.Read(f, binary.LittleEndian, &frame); err != nil {
			log.WithError(err).Fatal("Couldn't read frame")
		}

		frames = append(frames, frame)
	}

	sessions := make([]*discordgo.Session, SESSIONS)

	for i := 0; i < SESSIONS; i++ {
		session, err := discordgo.New(token)
		if err != nil {
			log.WithError(err).Fatal("Couldn't create Discord session")
		}
		session.LogLevel = discordgo.LogDebug

		ready := make(chan interface{})
		session.AddHandler(func(session *discordgo.Session, e *discordgo.Ready) {
			log.Info("Ready!")
			close(ready)
		})

		if err := session.Open(); err != nil {
			log.WithError(err).Fatal("Couldn't connect to Discord!")
		}

		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			log.Fatal("Took too long connecting to Discord")
		}

		sessions = append(sessions, session)

	}

	for {
		num := rand.Intn(len(sessions))
		log.WithFields(log.Fields{"sessions": len(sessions), "chosen": num}).Info("Running")
		session := sessions[num]

		vc, err := session.ChannelVoiceJoin(gid, cid, false, false)
		if err != nil {
			log.Fatal("Couldn't join voice channel")
		}
		vc.Speaking(true)
		for _, frame := range frames {
			vc.OpusSend <- frame
		}
		vc.Speaking(false)

		if err := vc.Disconnect(); err != nil {
			log.WithError(err).Fatal("Couldn't leave voice channel")
		}
	}

	return nil
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "token, t",
			EnvVar: "TOKEN",
		},
		cli.StringFlag{
			Name:   "guild, g",
			EnvVar: "GUILD",
		},
		cli.StringFlag{
			Name:   "channel, c",
			EnvVar: "CHANNEL",
		},
	}
	app.Action = action
	app.Run(os.Args)
}
