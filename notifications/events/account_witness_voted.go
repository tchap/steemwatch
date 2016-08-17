package events

import (
	"github.com/go-steem/rpc/apis/database"
)

type AccountWitnessVoted struct {
	Op *database.AccountWitnessVoteOperation
}

type AccountWitnessVotedEventMiner struct{}

func NewAccountWitnessVotedEventMiner() *AccountWitnessVotedEventMiner {
	return &AccountWitnessVotedEventMiner{}
}

func (miner *AccountWitnessVotedEventMiner) MineEvent(
	operation *database.Operation,
	content *database.Content, // nil
) ([]interface{}, error) {

	op, ok := operation.Body.(*database.AccountWitnessVoteOperation)
	if !ok {
		return nil, nil
	}
	return []interface{}{&AccountWitnessVoted{op}}, nil
}
