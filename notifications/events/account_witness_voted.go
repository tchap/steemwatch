package events

import (
	"github.com/go-steem/rpc/apis/database"
	"github.com/go-steem/rpc/types"
)

type AccountWitnessVoted struct {
	Op *types.AccountWitnessVoteOperation
}

type AccountWitnessVotedEventMiner struct{}

func NewAccountWitnessVotedEventMiner() *AccountWitnessVotedEventMiner {
	return &AccountWitnessVotedEventMiner{}
}

func (miner *AccountWitnessVotedEventMiner) MineEvent(
	operation types.Operation,
	content *database.Content, // nil
) ([]interface{}, error) {

	op, ok := operation.Data().(*types.AccountWitnessVoteOperation)
	if !ok {
		return nil, nil
	}
	return []interface{}{&AccountWitnessVoted{op}}, nil
}
