// examples/discordgo-bot/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
	"github.com/bwmarrin/discordgo"
)

var (
	Token       string
	prefManager *userprefs.Manager
	guildID     string // For testing, remove for production
)

func init() {
	flag.StringVar(&Token, "t", "", "Bot Token")
	flag.StringVar(&guildID, "g", "", "Guild ID for testing")
	flag.Parse()
}

func main() {
	// Initialize SQLite storage
	store, err := storage.NewSQLiteStorage("preferences.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize memory cache
	memCache := cache.NewMemoryCache()
	defer memCache.Close()

	// Create preference manager
	prefManager = userprefs.New(
		userprefs.WithStorage(store),
		userprefs.WithCache(memCache),
	)

	// Define preferences
	preferences := []userprefs.PreferenceDefinition{
		{
			Key:          "default_format",
			Type:         "enum",
			Category:     "media",
			DefaultValue: "gif",
			AllowedValues: []interface{}{
				"gif", "mp4", "webp",
			},
		},
		{
			Key:          "auto_convert",
			Type:         "boolean",
			Category:     "media",
			DefaultValue: false,
		},
		{
			Key:          "quality",
			Type:         "enum",
			Category:     "media",
			DefaultValue: "medium",
			AllowedValues: []interface{}{
				"low", "medium", "high",
			},
		},
	}

	// Register preferences
	for _, pref := range preferences {
		if err := prefManager.DefinePreference(pref); err != nil {
			log.Fatalf("Failed to define preference %s: %v", pref.Key, err)
		}
	}

	// Create Discord session
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	// Register handlers
	dg.AddHandler(ready)
	dg.AddHandler(interactionCreate)

	// Open websocket connection
	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}
	defer dg.Close()

	// Register commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "preferences",
			Description: "Manage your preferences",
			Type:        discordgo.ChatApplicationCommand,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "view",
					Description: "View your current preferences",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "set",
					Description: "Set a preference",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "format",
							Description: "Set default format",
							Type:        discordgo.ApplicationCommandOptionString,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "GIF",
									Value: "gif",
								},
								{
									Name:  "MP4",
									Value: "mp4",
								},
								{
									Name:  "WebP",
									Value: "webp",
								},
							},
						},
						{
							Name:        "auto_convert",
							Description: "Enable/disable auto conversion",
							Type:        discordgo.ApplicationCommandOptionBoolean,
						},
						{
							Name:        "quality",
							Description: "Set quality level",
							Type:        discordgo.ApplicationCommandOptionString,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{
									Name:  "Low",
									Value: "low",
								},
								{
									Name:  "Medium",
									Value: "medium",
								},
								{
									Name:  "High",
									Value: "high",
								},
							},
						},
					},
				},
			},
		},
	}

	// Register commands with Discord
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, cmd := range commands {
		rcmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, guildID, cmd)
		if err != nil {
			log.Panicf("Cannot create command %v: %v", cmd.Name, err)
		}
		registeredCommands[i] = rcmd
	}

	// Wait for interrupt signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Bot is running. Press Ctrl+C to exit.")
	<-stop

	// Cleanup commands
	for _, cmd := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, guildID, cmd.ID)
		if err != nil {
			log.Printf("Cannot delete command %v: %v", cmd.Name, err)
		}
	}
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
}

func interactionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		handlePreferences(s, i)
	}
}

func handlePreferences(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	userID := i.Member.User.ID

	switch options[0].Name {
	case "view":
		handleViewPreferences(s, i, userID)
	case "set":
		handleSetPreferences(s, i, userID)
	}
}

func handleViewPreferences(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) {
	ctx := context.Background()
	prefs, err := prefManager.GetAll(ctx, userID)
	if err != nil {
		respondError(s, i, "Failed to get preferences")
		return
	}

	// Build response message
	content := "Your current preferences:\n"
	for key, pref := range prefs {
		content += fmt.Sprintf("• %s: %v\n", key, pref.Value)
	}

	respond(s, i, content)
}

func handleSetPreferences(s *discordgo.Session, i *discordgo.InteractionCreate, userID string) {
	ctx := context.Background()
	options := i.ApplicationCommandData().Options[0].Options

	for _, opt := range options {
		var err error
		switch opt.Name {
		case "format":
			err = prefManager.Set(ctx, userID, "default_format", opt.StringValue())
		case "auto_convert":
			err = prefManager.Set(ctx, userID, "auto_convert", opt.BoolValue())
		case "quality":
			err = prefManager.Set(ctx, userID, "quality", opt.StringValue())
		}

		if err != nil {
			respondError(s, i, fmt.Sprintf("Failed to set %s preference", opt.Name))
			return
		}
	}

	respond(s, i, "Preferences updated successfully!")
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}

func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error responding to interaction: %v", err)
	}
}
