package diskplayer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func Play() error {
	p := GetConfigString(RECORD_PATH)
	u, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}
	return PlayUri(string(u))
}

func PlayUri(u string) error {
	if u == "" {
		return errors.New("Spotify URI is required.")
	}

	spotifyUri := spotify.URI(u)

	c, err := client()
	if err != nil {
		return err
	}

	d := GetConfigString(SPOTIFY_DEVICE_NAME)
	id, err := getPlayerId(c, d)
	if err != nil {
		return err
	}

	o := &spotify.PlayOptions{
		DeviceID:        id,
		PlaybackContext: &spotifyUri,
	}

	return c.PlayOpt(o)
}

func Pause() error {
	c, err := client()
	if err != nil {
		return err
	}
	return c.Pause()
}

func client() (*spotify.Client, error) {

	var s *http.Server
	ch := make(chan *spotify.Client, 1)

	t, err := tokenFromFile()
	if err != nil {
		s, _ = fetchNewToken(ch)
	} else {
		auth, err := newAuthenticator()
		if err != nil {
			return nil, err
		}
		c := auth.NewClient(t)
		ch <- &c
	}

	c := <-ch

	if s != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := s.Shutdown(ctx)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func fetchNewToken(ch chan *spotify.Client) (*http.Server, error) {
	auth, err := newAuthenticator()
	if err != nil {
		return nil, err
	}
	h := CallbackHandler{ch: ch, auth: auth}
	s := RunCallbackServer(h)
	u := auth.AuthURL(STATE_IDENTIFIER)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", u)
	return s, nil
}

func newAuthenticator() (*spotify.Authenticator, error) {
	r := GetConfigString(SPOTIFY_CALLBACK_URL)
	u, err := url.Parse(r)
	if err != nil {
		return nil, err
	}

	id := GetConfigString(SPOTIFY_CLIENT_ID)
	s := GetConfigString(SPOTIFY_CLIENT_SECRET)

	// Unset any existing environment variables
	err = os.Unsetenv(SPOTIFY_ID_ENV_VAR)
	if err != nil {
		return nil, err
	}
	err = os.Unsetenv(SPOTIFY_SECRET_ENV_VAR)
	if err != nil {
		return nil, err
	}

	// Set the environment variables required for Spotify auth
	err = os.Setenv(SPOTIFY_ID_ENV_VAR, id)
	if err != nil {
		return nil, err
	}
	err = os.Setenv(SPOTIFY_SECRET_ENV_VAR, s)
	if err != nil {
		return nil, err
	}

	auth := spotify.NewAuthenticator(u.String(), spotify.ScopeUserReadPrivate, spotify.ScopePlaylistReadPrivate,
		spotify.ScopeUserModifyPlaybackState, spotify.ScopeUserReadPlaybackState)

	return &auth, nil
}

type CallbackHandler struct {
	ch   chan *spotify.Client
	auth *spotify.Authenticator
}

func (h CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t, err := h.auth.Token(STATE_IDENTIFIER, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != STATE_IDENTIFIER {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, STATE_IDENTIFIER)
	}

	err = saveToken(t)
	if err != nil {
		log.Fatal(err)
	}

	c := h.auth.NewClient(t)
	fmt.Fprintf(w, "Login Completed!")
	h.ch <- &c
}

func getPlayerId(c *spotify.Client, n string) (*spotify.ID, error) {
	ds, err := c.PlayerDevices()
	if err != nil {
		return nil, err
	}

	var id *spotify.ID
	for _, d := range ds {
		if d.Name == n {
			id = &d.ID
			if !d.Active {
				err := c.Pause()
				if err != nil {
					return nil, err
				}
				err = c.TransferPlayback(*id, false)
				if err != nil {
					return nil, err
				}
			}
			break
		}
	}

	if id == nil {
		return nil, errors.New("Player not found")
	}

	return id, nil
}

// Retrieves a token from a local file.
func tokenFromFile() (*oauth2.Token, error) {
	p := GetConfigString(TOKEN_PATH)

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	return t, err
}

// Saves a token to a file path.
func saveToken(token *oauth2.Token) error {
	p := GetConfigString(TOKEN_PATH)

	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}