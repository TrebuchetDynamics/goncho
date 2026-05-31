package goncho

import (
	"context"

	fileimport "github.com/TrebuchetDynamics/goncho/internal/fileimport"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

// ImportFileParams is the local Goncho equivalent of Honcho's multipart file
// upload request body. Content is consumed in memory and is not persisted as
// original file bytes.
type ImportFileParams = fileimport.Params

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
type ImportedFileMessage = fileimport.Message

// FileImportMetadata mirrors Honcho's file-related internal metadata attached
// to every message generated from an uploaded document.
type FileImportMetadata = fileimport.Metadata

// ImportFile converts a text-like file into ordinary ready user turns for the
// requested session. The original uploaded bytes are only used for extraction.
func (s *Service) ImportFile(ctx context.Context, params ImportFileParams) (FileImportResult, error) {
	result, err := fileimport.Import(ctx, fileimport.Options{
		DB:                    s.db,
		WorkspaceID:           s.workspaceID,
		MaxFileSize:           s.maxFileSize,
		MaxMessageSize:        s.maxMessageSize,
		DefaultMaxMessageSize: DefaultMaxMessageSize,
	}, params)
	if err != nil {
		return FileImportResult{}, err
	}
	return FileImportResult{
		WorkspaceID: result.WorkspaceID,
		SessionKey:  result.SessionKey,
		PeerID:      result.PeerID,
		FileID:      result.FileID,
		Messages:    result.Messages,
		Unavailable: convertFileImportUnavailable(result.Unavailable),
	}, nil
}

func convertFileImportUnavailable(items []fileimport.UnavailableEvidence) []ContextUnavailableEvidence {
	if len(items) == 0 {
		return nil
	}
	return sliceutil.Map(items, func(item fileimport.UnavailableEvidence) ContextUnavailableEvidence {
		return ContextUnavailableEvidence{
			Field:      item.Field,
			Capability: item.Capability,
			Reason:     item.Reason,
		}
	})
}
