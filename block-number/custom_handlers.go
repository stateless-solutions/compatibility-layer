package blocknumber

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stateless-solutions/stateless-compatibility-layer/models"
)

type customHandlersHolder struct{}

func (customHandlersHolder) HandleGetLogsAndBlockRange(req *models.RPCReq) ([]*rpc.BlockNumberOrHash, error) {
	pos := 0

	var p []map[string]interface{}
	err := json.Unmarshal(req.Params, &p)
	if err != nil {
		return nil, err
	}

	if len(p) <= pos {
		return nil, ErrParseErr
	}

	block, err := remarshalTagMap(p[pos], "blockHash")
	if err != nil {
		return nil, err
	}
	if block != nil && block.BlockHash != nil {
		return []*rpc.BlockNumberOrHash{block}, nil // if block hash is set fromBlock and toBlock are ignored
	}

	fromBlock, err := remarshalTagMap(p[pos], "fromBlock")
	if err != nil {
		return nil, err
	}
	if fromBlock == nil || fromBlock.BlockNumber == nil {
		b := rpc.BlockNumberOrHashWithNumber(rpc.EarliestBlockNumber)
		fromBlock = &b
	}
	toBlock, err := remarshalTagMap(p[pos], "toBlock")
	if err != nil {
		return nil, err
	}
	if toBlock == nil || toBlock.BlockNumber == nil {
		b := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
		toBlock = &b
	}
	return []*rpc.BlockNumberOrHash{fromBlock, toBlock}, nil // always keep this order
}
