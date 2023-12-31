package synthesizer

import (
	"encoding/json"
	"fmt"
	"github.com/petrzlen/vocode-golang/pkg/models"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{}

type openAITTS struct {
	apiKey string
}

func NewOpenAITTS(openAIAPIKey string) Synthesizer {
	return &openAITTS{
		apiKey: openAIAPIKey,
	}
}

// TODO(devx, P1): Replace with the openai-go one after implemented
// https://github.com/sashabaranov/go-openai/pull/528/files?diff=unified&w=0
func (o *openAITTS) CreateSpeech(text string, speed float64) (audioOutput models.AudioData, err error) {
	model := "tts-1"
	// TODO(P0, ux): Experiment with this a bit for speed and quality
	responseFormat := "mp3"
	// responseFormat := "flac"
	// TODO(P2, ux): Opus should be a better format for streaming BUT I would probably painfully die making it work in Golang.

	log.Debug().Str("input", text).Float64("speed", speed).Str("output_format", responseFormat).Str("model", model).Msg("sendTTSRequest start")

	payload := TTSPayload{
		Model:          model,
		Input:          text,
		Voice:          "echo",
		ResponseFormat: responseFormat,
		Speed:          speed,
	}
	reqStr, _ := json.Marshal(payload)
	rawAudioBytes, err := o.sendRequest("POST", "audio/speech", string(reqStr))
	if err != nil {
		err = fmt.Errorf("could not do audio/speech for %s cause %w", reqStr, err)
		return
	}

	audioOutput = models.AudioData{
		ByteData: rawAudioBytes,
		Format:   responseFormat,
		Length:   0, // TODO
		Text:     text,
		Trace:    models.NewTrace("openAITTS.CreateSpeech"),
	}

	return
}

// TTSPayload for sendTTSRequest
type TTSPayload struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}

// This is to by-pass not-yet-implemented APIs in go-openai
func (o *openAITTS) sendRequest(method string, endpoint string, requestStr string) (result []byte, err error) {
	requestStart := time.Now()
	// Construct the request body
	reqBody := strings.NewReader(requestStr)

	// Create and send the request
	req, err := http.NewRequest(method, "https://api.openai.com/v1/"+endpoint, reqBody)
	if err != nil {
		return
	}
	req.Header.Add("Authorization", "Bearer "+o.apiKey)
	req.Header.Add("Content-Type", "application/json")

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer func() { resp.Body.Close() }()

	log.Debug().Dur("request_time", time.Since(requestStart)).Str("method", method).Str("endpoint", endpoint).Int("status_code", resp.StatusCode).Msg("request done")

	if resp.StatusCode != http.StatusOK {
		errMsg, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("received non-200 status %d from %s: %s", resp.StatusCode, endpoint, errMsg)
		log.Debug().Err(err).Str("method", method).Str("endpoint", endpoint).Str("requestStr", requestStr).Msg("request to openai failed")
		return
	}

	readStart := time.Now()
	result, err = io.ReadAll(resp.Body)
	log.Debug().Dur("response_body_read_time", time.Since(readStart)).Int("response_byte_size", len(result)).Str("endpoint", endpoint).Msg("request body read done")
	if err != nil {
		err = fmt.Errorf("could not read response %w", err)
		return
	}
	return
}
