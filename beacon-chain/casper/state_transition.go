package casper

/*
import (
	"encoding/binary"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ApplyCrosslinkRewardsAndPenalties applies the appropriate rewards and penalties according to the attestation
// for a shard.
func ApplyCrosslinkRewardsAndPenalties(
	crosslinkRecords []*pb.CrosslinkRecord,
	slot uint64,
	attesterIndices []uint32,
	attestation *pb.AggregatedAttestation,
	validators []*pb.ValidatorRecord,
	totalBalance uint64,
	voteBalance uint64) error {

	rewardQuotient := RewardQuotient(validators)

	for _, attesterIndex := range attesterIndices {
		timeSinceLastConfirmation := slot - crosslinkRecords[attestation.Shard].GetSlot()

		if !crosslinkRecords[attestation.Shard].GetRecentlyChanged() {
			checkBit, err := bitutil.CheckBit(attestation.AttesterBitfield, int(attesterIndex))
			if err != nil {
				return err
			}

			if checkBit {
				RewardValidatorCrosslink(totalBalance, voteBalance, rewardQuotient, validators[attesterIndex])
			} else {
				PenaliseValidatorCrosslink(timeSinceLastConfirmation, rewardQuotient, validators[attesterIndex])
			}
		}
	}
	return nil
}

// ProcessBalancesInCrosslink checks the vote balances and if there is a supermajority it sets the crosslink
// for that shard.
func ProcessBalancesInCrosslink(slot uint64, voteBalance uint64, totalBalance uint64,
	attestation *pb.AggregatedAttestation, crosslinkRecords []*pb.CrosslinkRecord) []*pb.CrosslinkRecord {

	// if 2/3 of committee voted on this crosslink, update the crosslink
	// with latest dynasty number, shard block hash, and slot number.

	voteMajority := 3*voteBalance >= 2*totalBalance
	if voteMajority && !crosslinkRecords[attestation.Shard].RecentlyChanged {
		crosslinkRecords[attestation.Shard] = &pb.CrosslinkRecord{
			RecentlyChanged: true,
			ShardBlockHash:  attestation.ShardBlockHash,
			Slot:            slot,
		}
	}
	return crosslinkRecords
}
*/
