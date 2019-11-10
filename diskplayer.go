package diskplayer

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Play will play an album or playlist by reading a Spotify URI from a file whose filepath is defined in the
// diskplayer.yaml configuration file under the recorder.file_path entry.
// An error is returned if one is encountered.
func Play() error {
	p := ConfigValue(RECORD_PATH)
	return PlayPath(p)
}

// PlayPath will play an album or playlist by reading a Spotify URI from a file whose filepath is passed into the
// function.
// An error is returned if one is encountered.
func PlayPath(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	var l string
	for s.Scan() {
		l = s.Text()
		break // only interested in one line
	}

	if l == "" {
		return fmt.Errorf("unable to read line from path: %s", p)
	}

	return PlayUri(string(l))
}

// PlayURI will play the album or playlist Spotify URI that is passed in to the function.
// An error is returned if one is encountered.
func PlayUri(u string) error {
	if u == "" {
		return errors.New("spotify URI is required")
	}

	spotifyUri := spotify.URI(u)

	c, err := client()
	if err != nil {
		return err
	}

	n := ConfigValue(SPOTIFY_DEVICE_NAME)
	ds, err := c.PlayerDevices()
	if err != nil {
		return err
	}

	activeID := activePlayerId(&ds)
	if activeID == nil {
		return nil
	}

	playerID := diskplayerId(&ds, n)
	if playerID == nil {
		return fmt.Errorf("client identified by %s not found", n)
	}

	if *activeID != *playerID {
		err := c.Pause()
		if err != nil {
			return err
		}
		err = c.TransferPlayback(*playerID, false)
		if err != nil {
			return err
		}
	}

	o := &spotify.PlayOptions{
		DeviceID:        playerID,
		PlaybackContext: &spotifyUri,
	}

	return c.PlayOpt(o)
}

// Pause will pause the Spotify playback if the Diskplayer is the currently active Spotify device.
// An error is returned if one is encountered.
func Pause() error {
	c, err := client()
	if err != nil {
		return err
	}

	n := ConfigValue(SPOTIFY_DEVICE_NAME)
	ds, err := c.PlayerDevices()
	if err != nil {
		return err
	}

	activeID := activePlayerId(&ds)
	if activeID == nil {
		return nil
	}

	playerID := diskplayerId(&ds, n)
	if playerID == nil {
		return fmt.Errorf("client identified by %s not found", n)
	}

	if *activeID == *playerID {
		err := c.Pause()
		if err != nil {
			return err
		}
	}

	return nil
}

// Returns an authenticated Spotify client object, or an error if encountered.
func client() (*spotify.Client, error) {

	var s *http.Server
	ch := make(chan *spotify.Client, 1)

	t, err := tokenFromFile()
	if err != nil {
		if err == err.(*os.PathError) {
			s, _ = fetchNewToken(ch)
		} else {
			return nil, err
		}
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

// fetchNewToken will prompt the user to log in to Spotify to obtain a new authentication token.
// Returns the server object which is waiting for the callback request or any error encountered.
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

// newAuthenticator returns a new Spotify client authenticator object configured using the values specified in the
// diskplayer.yaml configuration file.
// Returns a new Spotify Authenticator object or any error encountered.
func newAuthenticator() (*spotify.Authenticator, error) {
	r := ConfigValue(SPOTIFY_CALLBACK_URL)
	u, err := url.Parse(r)
	if err != nil {
		return nil, err
	}

	id := ConfigValue(SPOTIFY_CLIENT_ID)
	s := ConfigValue(SPOTIFY_CLIENT_SECRET)

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

// An implementation of the Handler ServeHTTP function for the CallbackHandler struct.
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

// activePlayerIds iterates through the provided player devices and returns the active ID. If there is no active
// Spotify client device the ID will be returned as a nil pointer.
func activePlayerId(ds *[]spotify.PlayerDevice) *spotify.ID {
	var id *spotify.ID
	for _, d := range *ds {
		if d.Active {
			id = &d.ID
		}
	}

	return id
}

// diskplayerId returns the Spotify ID for the Spotify client whose name is provided in the parameter list,
// or a nil pointer if no matching device is found.
func diskplayerId(ds *[]spotify.PlayerDevice, n string) *spotify.ID {
	var id *spotify.ID
	for _, d := range *ds {
		if d.Name == n {
			id = &d.ID
		}
	}

	return id
}

// tokenFromFile will attempt to deserialize a token whose path is defined in the diskplayer.
// yaml configuration file under the token.file_path field.
// Returns a pointer to an oauth2 token object or any error encountered.
func tokenFromFile() (*oauth2.Token, error) {
	p := ConfigValue(TOKEN_PATH)

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	return t, err
}

// saveToken will serialize the provided token and save it to the file whose path is defined in the diskplayer.
// yaml configuration file under the token.file_path field.
// Returns an error if one is encountered.
func saveToken(token *oauth2.Token) error {
	p := ConfigValue(TOKEN_PATH)

	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}
