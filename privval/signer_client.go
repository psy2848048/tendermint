package privval

import (
	"time"

	"github.com/pkg/errors"

	"github.com/hdac-io/tendermint/crypto"
	"github.com/hdac-io/tendermint/types"
)

// SignerClient implements PrivValidator.
// Handles remote validator connections that provide signing services
type SignerClient struct {
	endpoint *SignerListenerEndpoint
}

var _ types.PrivValidator = (*SignerClient)(nil)

// NewSignerClient returns an instance of SignerClient.
// it will start the endpoint (if not already started)
func NewSignerClient(endpoint *SignerListenerEndpoint) (*SignerClient, error) {
	if !endpoint.IsRunning() {
		if err := endpoint.Start(); err != nil {
			return nil, errors.Wrap(err, "failed to start listener endpoint")
		}
	}

	return &SignerClient{endpoint: endpoint}, nil
}

// Close closes the underlying connection
func (sc *SignerClient) Close() error {
	return sc.endpoint.Close()
}

// IsConnected indicates with the signer is connected to a remote signing service
func (sc *SignerClient) IsConnected() bool {
	return sc.endpoint.IsConnected()
}

// WaitForConnection waits maxWait for a connection or returns a timeout error
func (sc *SignerClient) WaitForConnection(maxWait time.Duration) error {
	return sc.endpoint.WaitForConnection(maxWait)
}

// SetImmutableHeight remove signature lower than target height(usage: last commited height)
// Implements ParallelProgressablePV
func (sc *SignerClient) SetImmutableHeight(height int64) error {
	_, err := sc.endpoint.SendRequest(&SetImmutableHeightRequest{height})
	if err != nil {
		sc.endpoint.Logger.Error("SignerClient::SetImmutableHeight", "err", err)
	}
	return err
}

//--------------------------------------------------------
// Implement PrivValidator

// Ping sends a ping request to the remote signer
func (sc *SignerClient) Ping() error {
	response, err := sc.endpoint.SendRequest(&PingRequest{})

	if err != nil {
		sc.endpoint.Logger.Error("SignerClient::Ping", "err", err)
		return nil
	}

	_, ok := response.(*PingResponse)
	if !ok {
		sc.endpoint.Logger.Error("SignerClient::Ping", "err", "response != PingResponse")
		return err
	}

	return nil
}

// GetPubKey retrieves a public key from a remote signer
func (sc *SignerClient) GetPubKey() crypto.PubKey {
	response, err := sc.endpoint.SendRequest(&PubKeyRequest{})
	if err != nil {
		sc.endpoint.Logger.Error("SignerClient::GetPubKey", "err", err)
		return nil
	}

	pubKeyResp, ok := response.(*PubKeyResponse)
	if !ok {
		sc.endpoint.Logger.Error("SignerClient::GetPubKey", "err", "response != PubKeyResponse")
		return nil
	}

	if pubKeyResp.Error != nil {
		sc.endpoint.Logger.Error("failed to get private validator's public key", "err", pubKeyResp.Error)
		return nil
	}

	return pubKeyResp.PubKey
}

// SignVote requests a remote signer to sign a vote
func (sc *SignerClient) SignVote(chainID string, vote *types.Vote) error {
	response, err := sc.endpoint.SendRequest(&SignVoteRequest{Vote: vote})
	if err != nil {
		sc.endpoint.Logger.Error("SignerClient::SignVote", "err", err)
		return err
	}

	resp, ok := response.(*SignedVoteResponse)
	if !ok {
		sc.endpoint.Logger.Error("SignerClient::GetPubKey", "err", "response != SignedVoteResponse")
		return ErrUnexpectedResponse
	}

	if resp.Error != nil {
		return resp.Error
	}
	*vote = *resp.Vote

	return nil
}

// SignProposal requests a remote signer to sign a proposal
func (sc *SignerClient) SignProposal(chainID string, proposal *types.Proposal) error {
	response, err := sc.endpoint.SendRequest(&SignProposalRequest{Proposal: proposal})
	if err != nil {
		sc.endpoint.Logger.Error("SignerClient::SignProposal", "err", err)
		return err
	}

	resp, ok := response.(*SignedProposalResponse)
	if !ok {
		sc.endpoint.Logger.Error("SignerClient::SignProposal", "err", "response != SignedProposalResponse")
		return ErrUnexpectedResponse
	}
	if resp.Error != nil {
		return resp.Error
	}
	*proposal = *resp.Proposal

	return nil
}

// GetParallelProgressablePV implements PrivValidator.
func (sc *SignerClient) GetParallelProgressablePV() types.ParallelProgressablePV {
	return sc
}
