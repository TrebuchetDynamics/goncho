package goncho

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

// ImportFileParams is the local Goncho equivalent of Honcho's multipart file
// upload request body. Content is consumed in memory and is not persisted as
// original file bytes.
type ImportFileParams struct {
	SessionKey    string         `json:"session_key"`
	PeerID        string         `json:"peer_id"`
	Filename      string         `json:"filename"`
	ContentType   string         `json:"content_type"`
	Content       []byte         `json:"-"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
	CreatedAt     *time.Time     `json:"created_at,omitempty"`
}

// FileImportResult describes the ordinary session messages written from an
// import plus degraded-mode evidence for reasoning work that cannot be queued.
type FileImportResult struct {
	WorkspaceID string                       `json:"workspace_id"`
	SessionKey  string                       `json:"session_key"`
	PeerID      string                       `json:"peer_id"`
	FileID      string                       `json:"file_id"`
	Messages    []ImportedFileMessage        `json:"messages"`
	Unavailable []ContextUnavailableEvidence `json:"unavailable,omitempty"`
}

// ImportedFileMessage is the stable return shape for each imported chunk.
type ImportedFileMessage struct {
	ID            int64              `json:"id"`
	SessionKey    string             `json:"session_key"`
	PeerID        string             `json:"peer_id"`
	Role          string             `json:"role"`
	Content       string             `json:"content"`
	CreatedAt     time.Time          `json:"created_at"`
	Metadata      map[string]any     `json:"metadata,omitempty"`
	Configuration map[string]any     `json:"configuration,omitempty"`
	File          FileImportMetadata `json:"file"`
}

// FileImportMetadata mirrors Honcho's file-related internal metadata attached
// to every message generated from an uploaded document.
type FileImportMetadata struct {
	FileID              string `json:"file_id"`
	Filename            string `json:"filename"`
	ChunkIndex          int    `json:"chunk_index"`
	TotalChunks         int    `json:"total_chunks"`
	OriginalFileSize    int64  `json:"original_file_size"`
	ContentType         string `json:"content_type"`
	ChunkCharacterRange [2]int `json:"chunk_character_range"`
}

type fileChunk struct {
	content string
	start   int
	end     int
}

// ImportFile converts a text-like file into ordinary ready user turns for the
// requested session. The original uploaded bytes are only used for extraction.
func (s *Service) ImportFile(ctx context.Context, params ImportFileParams) (FileImportResult, error) {
	sessionKey := strings.TrimSpace(params.SessionKey)
	if sessionKey == "" {
		return FileImportResult{}, fmt.Errorf("goncho: session_key is required")
	}
	peerID := strings.TrimSpace(params.PeerID)
	if peerID == "" {
		return FileImportResult{}, fmt.Errorf("goncho: peer_id is required")
	}
	contentType := normalizeContentType(params.ContentType)
	if contentType == "" {
		return FileImportResult{}, fmt.Errorf("goncho: content_type is required")
	}
	if s.maxFileSize > 0 && len(params.Content) > s.maxFileSize {
		return FileImportResult{}, fmt.Errorf("goncho: file size %d exceeds maximum %d", len(params.Content), s.maxFileSize)
	}

	text, err := extractImportText(contentType, params.Content)
	if err != nil {
		return FileImportResult{}, err
	}
	maxChars := s.maxMessageSize
	if maxChars <= 0 {
		maxChars = DefaultMaxMessageSize
	}
	chunks := splitImportTextIntoChunks(text, maxChars)
	if len(chunks) == 0 {
		return FileImportResult{}, errors.New("goncho: file import produced no messages")
	}

	fileID, err := newImportFileID()
	if err != nil {
		return FileImportResult{}, err
	}
	createdAt := time.Now().UTC()
	if params.CreatedAt != nil {
		createdAt = params.CreatedAt.UTC()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FileImportResult{}, fmt.Errorf("goncho: begin file import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	messages := make([]ImportedFileMessage, 0, len(chunks))
	for i, chunk := range chunks {
		fileMeta := FileImportMetadata{
			FileID:              fileID,
			Filename:            params.Filename,
			ChunkIndex:          i,
			TotalChunks:         len(chunks),
			OriginalFileSize:    int64(len(params.Content)),
			ContentType:         contentType,
			ChunkCharacterRange: [2]int{chunk.start, chunk.end},
		}
		metaJSON, err := marshalImportMeta(fileMeta, params.Metadata, params.Configuration)
		if err != nil {
			return FileImportResult{}, err
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json, memory_sync_status)
			VALUES(?, 'user', ?, ?, ?, ?, 'ready')
		`, sessionKey, chunk.content, createdAt.Unix(), peerID, metaJSON)
		if err != nil {
			return FileImportResult{}, fmt.Errorf("goncho: insert imported file message: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return FileImportResult{}, fmt.Errorf("goncho: imported file message id: %w", err)
		}
		messages = append(messages, ImportedFileMessage{
			ID:            id,
			SessionKey:    sessionKey,
			PeerID:        peerID,
			Role:          "user",
			Content:       chunk.content,
			CreatedAt:     time.Unix(createdAt.Unix(), 0).UTC(),
			Metadata:      cloneMap(params.Metadata),
			Configuration: cloneMap(params.Configuration),
			File:          fileMeta,
		})
	}
	if err := tx.Commit(); err != nil {
		return FileImportResult{}, fmt.Errorf("goncho: commit file import: %w", err)
	}

	return FileImportResult{
		WorkspaceID: s.workspaceID,
		SessionKey:  sessionKey,
		PeerID:      peerID,
		FileID:      fileID,
		Messages:    messages,
		Unavailable: []ContextUnavailableEvidence{queueUnavailableEvidence()},
	}, nil
}

func normalizeContentType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err == nil {
		return strings.ToLower(mediaType)
	}
	return value
}

func extractImportText(contentType string, content []byte) (string, error) {
	switch {
	case contentType == "application/json":
		return extractJSONImportText(content)
	case strings.HasPrefix(contentType, "text/"):
		return decodeTextImportContent(content)
	default:
		return "", fmt.Errorf("goncho: unsupported content type %q", contentType)
	}
}

func extractJSONImportText(content []byte) (string, error) {
	if !utf8.Valid(content) {
		return "", errors.New("goncho: JSON uploads must be UTF-8 encoded")
	}
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return "", nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return "", fmt.Errorf("goncho: uploaded JSON is invalid: %w", err)
	}
	raw, err := json.Marshal(decoded)
	if err != nil {
		return "", fmt.Errorf("goncho: encode imported JSON text: %w", err)
	}
	return string(raw), nil
}

func decodeTextImportContent(content []byte) (string, error) {
	if utf8.Valid(content) {
		return string(content), nil
	}
	if decoded, ok := decodeUTF16WithBOM(content); ok {
		return decoded, nil
	}
	runes := make([]rune, len(content))
	for i, b := range content {
		runes[i] = rune(b)
	}
	return string(runes), nil
}

func decodeUTF16WithBOM(content []byte) (string, bool) {
	if len(content) < 2 {
		return "", false
	}
	littleEndian := false
	switch {
	case content[0] == 0xff && content[1] == 0xfe:
		littleEndian = true
	case content[0] == 0xfe && content[1] == 0xff:
		littleEndian = false
	default:
		return "", false
	}
	body := content[2:]
	if len(body)%2 != 0 {
		body = body[:len(body)-1]
	}
	u16 := make([]uint16, 0, len(body)/2)
	for i := 0; i < len(body); i += 2 {
		var value uint16
		if littleEndian {
			value = uint16(body[i]) | uint16(body[i+1])<<8
		} else {
			value = uint16(body[i])<<8 | uint16(body[i+1])
		}
		u16 = append(u16, value)
	}
	return string(utf16.Decode(u16)), true
}

func splitImportTextIntoChunks(text string, maxChars int) []fileChunk {
	runes := []rune(text)
	if len(runes) <= maxChars {
		return []fileChunk{{content: text, start: 0, end: len(runes)}}
	}

	var chunks []fileChunk
	current := 0
	for current < len(runes) {
		end := current + maxChars
		if end >= len(runes) {
			chunks = append(chunks, fileChunk{
				content: string(runes[current:]),
				start:   current,
				end:     len(runes),
			})
			break
		}
		breakPos := bestImportChunkBreak(runes, current, end)
		chunks = append(chunks, fileChunk{
			content: string(runes[current:breakPos]),
			start:   current,
			end:     breakPos,
		})
		current = breakPos
	}
	return chunks
}

func bestImportChunkBreak(runes []rune, start, end int) int {
	for _, delimiter := range []string{"\n\n", "\n", ". ", " "} {
		if pos := lastDelimiterRuneIndex(runes, delimiter, start, end); pos > start {
			return pos + len([]rune(delimiter))
		}
	}
	return end
}

func lastDelimiterRuneIndex(runes []rune, delimiter string, start, end int) int {
	needle := []rune(delimiter)
	if len(needle) == 0 || end-start < len(needle) {
		return -1
	}
	for i := end - len(needle); i >= start; i-- {
		if equalRunes(runes[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func equalRunes(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func marshalImportMeta(file FileImportMetadata, metadata, configuration map[string]any) (string, error) {
	meta := map[string]any{
		"file_id":               file.FileID,
		"filename":              file.Filename,
		"chunk_index":           file.ChunkIndex,
		"total_chunks":          file.TotalChunks,
		"original_file_size":    file.OriginalFileSize,
		"content_type":          file.ContentType,
		"chunk_character_range": []int{file.ChunkCharacterRange[0], file.ChunkCharacterRange[1]},
	}
	if metadata != nil {
		meta["metadata"] = cloneMap(metadata)
	}
	if configuration != nil {
		meta["configuration"] = cloneMap(configuration)
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("goncho: marshal file import metadata: %w", err)
	}
	return string(raw), nil
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		out := make(map[string]any, len(in))
		for k, v := range in {
			out[k] = v
		}
		return out
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		out = make(map[string]any, len(in))
		for k, v := range in {
			out[k] = v
		}
	}
	return out
}

func queueUnavailableEvidence() ContextUnavailableEvidence {
	return ContextUnavailableEvidence{
		Field:      "queue",
		Capability: "goncho_reasoning_queue",
		Reason:     "Goncho reasoning queue is unavailable; imported messages were written synchronously and are immediately visible as session messages",
	}
}

func newImportFileID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("goncho: generate file import id: %w", err)
	}
	var b bytes.Buffer
	b.WriteString("file_")
	b.WriteString(hex.EncodeToString(id[:]))
	return b.String(), nil
}
