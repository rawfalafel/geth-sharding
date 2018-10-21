package casper

/*
import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "casper")

// RewardValidatorCrosslink applies rewards to validators part of a shard committee for voting on a shard.
// TODO(#538): Change this to big.Int as tests using 64 bit integers fail due to integer overflow.
func RewardValidatorCrosslink(totalDeposit uint64, participatedDeposits uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) {
	currentBalance := int64(validator.Balance)
	currentBalance += int64(currentBalance) / int64(rewardQuotient) * (2*int64(participatedDeposits) - int64(totalDeposit)) / int64(totalDeposit)
	validator.Balance = uint64(currentBalance)
}

// PenaliseValidatorCrosslink applies penalties to validators part of a shard committee for not voting on a shard.
func PenaliseValidatorCrosslink(timeSinceLastConfirmation uint64, rewardQuotient uint64, validator *pb.ValidatorRecord) {
	newBalance := validator.Balance
	quadraticQuotient := quadraticPenaltyQuotient()
	newBalance -= newBalance/rewardQuotient + newBalance*timeSinceLastConfirmation/quadraticQuotient
	validator.Balance = newBalance
}
*/
