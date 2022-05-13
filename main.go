package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"bytes"
	"encoding/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/leighmurray/omedb"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

func main() {
	flag.Parse()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://api.hah.gay", "https://hah.gay"},
	}))

	r.Get("/record/start/{streamName}", func(w http.ResponseWriter, r *http.Request) {
		streamName := chi.URLParam(r, "streamName")
		client := &http.Client{}
		uniqueID := uuid.NewString()
		var jsonData = []byte(fmt.Sprintf(`{
			"id": "%s",
			"stream": {
				"name": "%s",
			}
		}`, uniqueID, streamName))

		req, _ := http.NewRequest("POST","http://127.0.0.1:8081/v1/vhosts/default/apps/app:startRecord", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic b21lLWFjY2Vzcy10b2tlbg==")

		resp, err := client.Do(req)

		if err != nil {
			panic("Couldn't access OME")
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic("Couldn't read the body")
		}
		w.Write([]byte(body))
	})

	r.Get("/record/stop/{recordID}", func(w http.ResponseWriter, r *http.Request) {
                recordID := chi.URLParam(r, "recordID")
                client := &http.Client{}
                var jsonData = []byte(fmt.Sprintf(`{
                        "id": "%s",
                }`, recordID))

                req, _ := http.NewRequest("POST","http://127.0.0.1:8081/v1/vhosts/default/apps/app:stopRecord", bytes.NewBuffer(jsonData))
                req.Header.Set("Content-Type", "application/json")
                req.Header.Set("Authorization", "Basic b21lLWFjY2Vzcy10b2tlbg==")

                resp, err := client.Do(req)

                if err != nil {
                        panic("Couldn't access OME")
                }
                defer resp.Body.Close()
                body, err := io.ReadAll(resp.Body)
                if err != nil {
                        panic("Couldn't read the body")
                }
                w.Write([]byte(body))
        })

	r.Get("/record/view/", func(w http.ResponseWriter, r *http.Request) {
                client := &http.Client{}

                req, _ := http.NewRequest("POST","http://127.0.0.1:8081/v1/vhosts/default/apps/app:records", nil)
                req.Header.Set("Content-Type", "application/json")
                req.Header.Set("Authorization", "Basic b21lLWFjY2Vzcy10b2tlbg==")

                resp, err := client.Do(req)

                if err != nil {
                        panic("Couldn't access OME")
                }
                defer resp.Body.Close()
                body, err := io.ReadAll(resp.Body)
                if err != nil {
                        panic("Couldn't read the body")
                }
                w.Write([]byte(body))
        })

	r.Get("/streams/", func(w http.ResponseWriter, r *http.Request) {
		client := &http.Client{}

		req, _ := http.NewRequest("GET", "http://127.0.0.1:8081/v1/vhosts/default/apps/app/streams", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Basic b21lLWFjY2Vzcy10b2tlbg==")

                resp, err := client.Do(req)

		if err != nil {
                        panic("Couldn't access OME")
                }
                defer resp.Body.Close()
                body, err := io.ReadAll(resp.Body)
                if err != nil {
                        panic("Couldn't read the body")
                }
                w.Write([]byte(body))
	})

	r.Post("/subscribe", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			panic("Couldn't read the body");
		}
		subscriptionObject := &webpush.Subscription{}
		json.Unmarshal(body, subscriptionObject)
		fmt.Println(subscriptionObject)
		omedb.AddSubscription(*subscriptionObject)
	})

	http.ListenAndServe(":8888", r)
}

// This is entirely optional, but I wanted to demonstrate how you could easily
// add your own logic to the render.Respond method.
func init() {
	render.Respond = func(w http.ResponseWriter, r *http.Request, v interface{}) {
		if err, ok := v.(error); ok {

			// We set a default error status response code if one hasn't been set.
			if _, ok := r.Context().Value(render.StatusCtxKey).(int); !ok {
				w.WriteHeader(400)
			}

			// We log the error
			fmt.Printf("Logging err: %s\n", err.Error())

			// We change the response to not reveal the actual error message,
			// instead we can transform the message something more friendly or mapped
			// to some code / language, etc.
			render.DefaultResponder(w, r, render.M{"status": "error"})
			return
		}

		render.DefaultResponder(w, r, v)
	}
}

// ErrResponse renderer type for handling all sorts of errors.
//
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

func ErrRender(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 422,
		StatusText:     "Error rendering response.",
		ErrorText:      err.Error(),
	}
}

var ErrNotFound = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}

