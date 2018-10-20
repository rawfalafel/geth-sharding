package types

import (
	"encoding/binary"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

var shardCount = params.GetConfig().ShardCount

// CrystallizedState contains fields of every Slot state,
// it changes every Slot.
type CrystallizedState struct {
	data *pb.CrystallizedState
}

// NewCrystallizedState creates a new crystallized state with a explicitly set data field.
func NewCrystallizedState(data *pb.CrystallizedState) *CrystallizedState {
	return &CrystallizedState{data: data}
}

// NewGenesisCrystallizedState initializes the crystallized state for slot 0.
func NewGenesisCrystallizedState(genesisValidators []*pb.ValidatorRecord) (*CrystallizedState, error) {
	// We seed the genesis crystallized state with a bunch of validators to
	// bootstrap the system.
	var err error
	if genesisValidators == nil {
		genesisValidators = casper.InitialValidators()

	}
	// Bootstrap attester indices for slots, each slot contains an array of attester indices.
	shardAndCommitteesForSlots, err := casper.InitialShardAndCommitteesForSlots(genesisValidators)
	if err != nil {
		return nil, err
	}

	// Bootstrap cross link records.
	var crosslinks []*pb.CrosslinkRecord
	for i := 0; i < shardCount; i++ {
		crosslinks = append(crosslinks, &pb.CrosslinkRecord{
			RecentlyChanged: false,
			ShardBlockHash:  make([]byte, 0, 32),
			Slot:            0,
		})
	}

	// Calculate total deposit from boot strapped validators.
	var totalDeposit uint64
	for _, v := range genesisValidators {
		totalDeposit += v.Balance
	}

	return &CrystallizedState{
		data: &pb.CrystallizedState{
			LastStateRecalculationSlot: 0,
			JustifiedStreak:            0,
			LastJustifiedSlot:          0,
			LastFinalizedSlot:          0,
			ValidatorSetChangeSlot:     0,
			Crosslinks:                 crosslinks,
			Validators:                 genesisValidators,
			ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
		},
	}, nil
}

// Proto returns the underlying protobuf data within a state primitive.
func (c *CrystallizedState) Proto() *pb.CrystallizedState {
	return c.data
}

// Marshal encodes crystallized state object into the wire format.
func (c *CrystallizedState) Marshal() ([]byte, error) {
	return proto.Marshal(c.data)
}

// Hash serializes the crystallized state object then uses
// blake2b to hash the serialized object.
func (c *CrystallizedState) Hash() ([32]byte, error) {
	data, err := proto.Marshal(c.data)
	if err != nil {
		return [32]byte{}, err
	}
	return hashutil.Hash(data), nil
}

// CopyState returns a deep copy of the current state.
func (c *CrystallizedState) CopyState() *CrystallizedState {
	crosslinks := make([]*pb.CrosslinkRecord, len(c.Crosslinks()))
	for index, crossLink := range c.Crosslinks() {
		crosslinks[index] = &pb.CrosslinkRecord{
			RecentlyChanged: crossLink.GetRecentlyChanged(),
			ShardBlockHash:  crossLink.GetShardBlockHash(),
			Slot:            crossLink.GetSlot(),
		}
	}

	validators := make([]*pb.ValidatorRecord, len(c.Validators()))
	for index, validator := range c.Validators() {
		validators[index] = &pb.ValidatorRecord{
			Pubkey:            validator.GetPubkey(),
			WithdrawalShard:   validator.GetWithdrawalShard(),
			WithdrawalAddress: validator.GetWithdrawalAddress(),
			RandaoCommitment:  validator.GetRandaoCommitment(),
			Balance:           validator.GetBalance(),
			Status:            validator.GetStatus(),
			ExitSlot:          validator.GetExitSlot(),
		}
	}

	shardAndCommitteesForSlots := make([]*pb.ShardAndCommitteeArray, len(c.ShardAndCommitteesForSlots()))
	for index, shardAndCommitteesForSlot := range c.ShardAndCommitteesForSlots() {
		shardAndCommittees := make([]*pb.ShardAndCommittee, len(shardAndCommitteesForSlot.GetArrayShardAndCommittee()))
		for index, shardAndCommittee := range shardAndCommitteesForSlot.GetArrayShardAndCommittee() {
			shardAndCommittees[index] = &pb.ShardAndCommittee{
				Shard:     shardAndCommittee.GetShard(),
				Committee: shardAndCommittee.GetCommittee(),
			}
		}
		shardAndCommitteesForSlots[index] = &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: shardAndCommittees,
		}
	}

	newState := CrystallizedState{&pb.CrystallizedState{
		LastStateRecalculationSlot: c.LastStateRecalculationSlot(),
		JustifiedStreak:            c.JustifiedStreak(),
		LastJustifiedSlot:          c.LastJustifiedSlot(),
		LastFinalizedSlot:          c.LastFinalizedSlot(),
		ValidatorSetChangeSlot:     c.ValidatorSetChangeSlot(),
		Crosslinks:                 crosslinks,
		Validators:                 validators,
		ShardAndCommitteesForSlots: shardAndCommitteesForSlots,
		DepositsPenalizedInPeriod:  c.DepositsPenalizedInPeriod(),
		ValidatorSetDeltaHashChain: c.data.ValidatorSetDeltaHashChain,
		PreForkVersion:             c.data.PreForkVersion,
		PostForkVersion:            c.data.PostForkVersion,
		ForkSlotNumber:             c.data.ForkSlotNumber,
	}}

	return &newState
}

// LastStateRecalculationSlot returns when the last time crystallized state recalculated.
func (c *CrystallizedState) LastStateRecalculationSlot() uint64 {
	return c.data.LastStateRecalculationSlot
}

// JustifiedStreak returns number of consecutive justified slots ending at head.
func (c *CrystallizedState) JustifiedStreak() uint64 {
	return c.data.JustifiedStreak
}

// LastJustifiedSlot return the last justified slot of the beacon chain.
func (c *CrystallizedState) LastJustifiedSlot() uint64 {
	return c.data.LastJustifiedSlot
}

// LastFinalizedSlot returns the last finalized Slot of the beacon chain.
func (c *CrystallizedState) LastFinalizedSlot() uint64 {
	return c.data.LastFinalizedSlot
}

// ValidatorSetChangeSlot returns the slot of last time validator set changes.
func (c *CrystallizedState) ValidatorSetChangeSlot() uint64 {
	return c.data.ValidatorSetChangeSlot
}

// ShardAndCommitteesForSlots returns the shard committee object.
func (c *CrystallizedState) ShardAndCommitteesForSlots() []*pb.ShardAndCommitteeArray {
	return c.data.ShardAndCommitteesForSlots
}

// Crosslinks returns the cross link records of the all the shards.
func (c *CrystallizedState) Crosslinks() []*pb.CrosslinkRecord {
	return c.data.Crosslinks
}

// Validators returns list of validators.
func (c *CrystallizedState) Validators() []*pb.ValidatorRecord {
	return c.data.Validators
}

// DepositsPenalizedInPeriod returns total deposits penalized in the given withdrawal period.
func (c *CrystallizedState) DepositsPenalizedInPeriod() []uint32 {
	return c.data.DepositsPenalizedInPeriod
}

// IsCycleTransition checks if a new cycle has been reached. At that point,
// a new crystallized state and active state transition will occur.
func (c *CrystallizedState) IsCycleTransition(slotNumber uint64) bool {
	return slotNumber >= c.LastStateRecalculationSlot()+params.GetConfig().CycleLength
}

// isValidatorSetChange checks if a validator set change transition can be processed. At that point,
// validator shuffle will occur.
func (c *CrystallizedState) isValidatorSetChange(slotNumber uint64) bool {
	if c.LastFinalizedSlot() <= c.ValidatorSetChangeSlot() {
		return false
	}
	if slotNumber-c.ValidatorSetChangeSlot() < params.GetConfig().MinValidatorSetChangeInterval {
		return false
	}

	shardProcessed := map[uint64]bool{}

	for _, shardAndCommittee := range c.ShardAndCommitteesForSlots() {
		for _, committee := range shardAndCommittee.ArrayShardAndCommittee {
			shardProcessed[committee.Shard] = true
		}
	}

	crosslinks := c.Crosslinks()
	for shard := range shardProcessed {
		if c.ValidatorSetChangeSlot() >= crosslinks[shard].Slot {
			return false
		}
	}
	return true
}

// getAttesterIndices fetches the attesters for a given attestation record.
func (c *CrystallizedState) getAttesterIndices(attestation *pb.AggregatedAttestation) ([]uint32, error) {
	slotsStart := c.LastStateRecalculationSlot() - params.GetConfig().CycleLength
	slotIndex := (attestation.Slot - slotsStart) % params.GetConfig().CycleLength
	return casper.CommitteeInShardAndSlot(slotIndex, attestation.GetShard(), c.data.GetShardAndCommitteesForSlots())
}

// CalculateNewState computes the new crystallized state, given the previous crystallized state
// and the current active state. This method is called during a cycle transition.
// We also check for validator set change transition and compute for new committees if necessary during this transition.
func (c *CrystallizedState) CalculateNewState(aState *ActiveState, block *Block) (*CrystallizedState, error) {
	newState := c.CopyState()

	var cycleStart uint64
	if c.LastStateRecalculationSlot() < params.GetConfig().CycleLength {
		cycleStart = 0
	} else {
		cycleStart = c.LastStateRecalculationSlot() - params.GetConfig().CycleLength
	}

	actives, totalBalance := getActivesAndTotal(c.Validators())

	// walk through all the slots from LastStateRecalculationSlot - cycleLength to LastStateRecalculationSlot - 1.
	newState.processBlocks(actives, aState, totalBalance, cycleStart)

	// TODO: Update balance based on original balance of validator, not balance after `CalculateRewards`.
	err := newState.processCrosslinks(aState.PendingAttestations(), actives, totalBalance)
	if err != nil {
		return nil, fmt.Errorf("unable to process crosslinks: %v", err)
	}

	// Process the pending special records gathered from last cycle.
	processSpecialRecords(actives, aState.PendingSpecials(), block.SlotNumber())

	// Exit the validators when their balance fall below min online deposit size.
	exitValidatorsBelowMinBalance(actives, block.SlotNumber())

	newState.data.LastStateRecalculationSlot = newState.LastStateRecalculationSlot() + params.GetConfig().CycleLength

	// Entering new validator set change transition.
	if newState.isValidatorSetChange(block.SlotNumber()) {
		log.Info("Entering validator set change transition")
		newState.data.ValidatorSetChangeSlot = newState.LastStateRecalculationSlot()
		newState.data.ShardAndCommitteesForSlots, err = newState.newValidatorSetRecalculations(block.ParentHash())
		if err != nil {
			return nil, err
		}

		period := block.SlotNumber() / params.GetConfig().WithdrawalPeriod
		totalPenalties := newState.penalizedETH(period)
		newState.data.Validators = casper.ChangeValidators(block.SlotNumber(), totalPenalties, totalBalance, newState.Validators())
	}

	return newState, nil
}

func (c *CrystallizedState) processBlocks(actives []*pb.ValidatorRecord, aState *ActiveState, totalBalance, cycleStart uint64) error {
	for i := uint64(0); i < params.GetConfig().CycleLength; i++ {
		slot := cycleStart + i
		blockHash := aState.RecentBlockHashes()[i]

		timeSinceFinality := slot - c.LastFinalizedSlot()
		// TODO: Handle no cache
		voteCache, _ := aState.blockVoteCache[blockHash]
		participatingTotal := voteCache.VoteTotalDeposit
		participatingIndices := voteCache.VoterIndices

		// TODO: Check if quadraticPenaltyQuotient should be in seconds or slots
		// TODO: Include penalties when `status == PENALIZED`
		calculateRewards(actives, timeSinceFinality, totalBalance, participatingTotal, participatingIndices)

		finalizeAndJustifySlots(c.Proto(), slot, participatingTotal, totalBalance)
	}

	return nil
}

// newValidatorSetRecalculations recomputes the validator set.
func (c *CrystallizedState) newValidatorSetRecalculations(seed [32]byte) ([]*pb.ShardAndCommitteeArray, error) {
	lastSlot := len(c.data.ShardAndCommitteesForSlots) - 1
	lastCommitteeFromLastSlot := len(c.ShardAndCommitteesForSlots()[lastSlot].ArrayShardAndCommittee) - 1
	crosslinkLastShard := c.ShardAndCommitteesForSlots()[lastSlot].ArrayShardAndCommittee[lastCommitteeFromLastSlot].Shard
	crosslinkNextShard := (crosslinkLastShard + 1) % uint64(shardCount)

	newShardCommitteeArray, err := casper.ShuffleValidatorsToCommittees(
		seed,
		c.data.Validators,
		crosslinkNextShard,
	)
	if err != nil {
		return nil, err
	}

	return append(c.data.ShardAndCommitteesForSlots[:params.GetConfig().CycleLength], newShardCommitteeArray...), nil
}

// processCrosslinks checks if the proposed shard block has recevied
// 2/3 of the votes. If yes, we update crosslink record to point to
// the proposed shard block with latest beacon chain slot numbers.
func (c *CrystallizedState) processCrosslinks(
	pendingAttestations []*pb.AggregatedAttestation,
	actives []*pb.ValidatorRecord,
	totalBalance uint64) error {

	rewardQuotient := getRewardQuotient(totalBalance)
	for _, attestation := range pendingAttestations {
		if c.Crosslinks()[attestation.Shard].RecentlyChanged {
			continue
		}

		indices, err := c.getAttesterIndices(attestation)
		if err != nil {
			return err
		}

		totalBalance, voteBalance, err := casper.AttestationBalances(actives, indices, attestation)
		if err != nil {
			return err
		}

		if 3*voteBalance < 2*totalBalance {
			continue
		}

		c.Crosslinks()[attestation.Shard] = &pb.CrosslinkRecord{
			RecentlyChanged: true,
			ShardBlockHash:  attestation.ShardBlockHash,
			Slot:            attestation.Slot,
		}

		timeSinceFinality := attestation.Slot - c.LastFinalizedSlot()
		rewardFn := getRewardFn(int64(voteBalance), int64(totalBalance), rewardQuotient)
		penaltyFn := getPenaltyFn(timeSinceFinality, rewardQuotient)

		for index, validatorIndex := range indices {
			attester := actives[validatorIndex]

			didVote, err := bitutil.CheckBit(attestation.AttesterBitfield, index)
			if err != nil {
				return fmt.Errorf("CheckBit failed: %v", err)
			}

			if didVote {
				rewardFn(attester)
			} else {
				penaltyFn(attester)
			}
		}
	}
	return nil
}

// penalizedETH calculates penalized total ETH during the last 3 withdrawal periods.
func (c *CrystallizedState) penalizedETH(periodIndex uint64) uint64 {
	var penalties uint64

	depositsPenalizedInPeriod := c.DepositsPenalizedInPeriod()
	penalties += uint64(depositsPenalizedInPeriod[periodIndex])

	if periodIndex >= 1 {
		penalties += uint64(depositsPenalizedInPeriod[periodIndex-1])
	}

	if periodIndex >= 2 {
		penalties += uint64(depositsPenalizedInPeriod[periodIndex-2])
	}

	return penalties
}

func getActivesAndTotal(validators []*pb.ValidatorRecord) ([]*pb.ValidatorRecord, uint64) {
	actives := make([]*pb.ValidatorRecord, 0, len(validators))
	total := uint64(0)
	for _, v := range validators {
		if v.Status == uint64(params.Active) {
			actives = append(actives, v)
			total += v.Balance
		}
	}

	return actives, total
}

func getRewardFn(participatingTotal, total int64, rewardQuotient uint64) func(*pb.ValidatorRecord) {
	return func(v *pb.ValidatorRecord) {
		balance := int64(v.Balance)
		participationProduct := participationProduct(participatingTotal, total)
		reward := balance / int64(rewardQuotient) * participationProduct
		v.Balance = uint64(balance + reward)
	}
}

func getPenaltyFn(timeSinceFinality, rewardQuotient uint64) func(*pb.ValidatorRecord) {
	return func(v *pb.ValidatorRecord) {
		balance := uint64(v.Balance)
		inactivityPenalty := balance / rewardQuotient
		finalityPenalty := balance * timeSinceFinality / quadraticPenaltyQuotient()
		v.Balance = balance - inactivityPenalty - finalityPenalty
	}
}

func getRewardQuotient(total uint64) uint64 {
	return params.GetConfig().BaseRewardQuotient * mathutil.IntegerSquareRoot(total)
}

func participationProduct(participatingTotal, total int64) int64 {
	return (2*participatingTotal - total) / total
}

func calculateRewards(actives []*pb.ValidatorRecord, timeSinceFinality, total, participatingTotal uint64, participatingIndices []uint32) {
	var rewardFn func(*pb.ValidatorRecord)
	var penaltyFn func(*pb.ValidatorRecord)

	rewardQuotient := getRewardQuotient(total)

	if timeSinceFinality <= 3*params.GetConfig().CycleLength {
		rewardFn = getRewardFn(int64(participatingTotal), int64(total), rewardQuotient)
		penaltyFn = func(v *pb.ValidatorRecord) {
			balance := int64(v.Balance)
			penalty := balance / int64(rewardQuotient)
			v.Balance = uint64(balance - penalty)
		}
	} else {
		rewardFn = func(v *pb.ValidatorRecord) {}
		penaltyFn = getPenaltyFn(timeSinceFinality, rewardQuotient)
	}

	updateBalance(actives, participatingIndices, rewardFn, penaltyFn)
}

func updateBalance(
	actives []*pb.ValidatorRecord,
	participatingIndices []uint32,
	reward func(*pb.ValidatorRecord),
	penalty func(*pb.ValidatorRecord)) {

	for index, v := range actives {
		if didParticipate(uint32(index), participatingIndices) {
			reward(v)
		} else {
			penalty(v)
		}
	}
}

func didParticipate(validatorIndex uint32, participatingIndices []uint32) bool {
	for _, i := range participatingIndices {
		if validatorIndex == i {
			return true
		}
	}
	return false
}

// finalizeAndJustifySlots justifies slots and sets the justified streak according to Casper FFG
// conditions. It also finalizes slots when the conditions are fulfilled.
func finalizeAndJustifySlots(cState *pb.CrystallizedState, slot, participatingTotal, total uint64) {
	// update lastJustifiedSlot
	if 3*participatingTotal >= 2*total {
		if slot > cState.LastJustifiedSlot {
			cState.LastJustifiedSlot = slot
		}
		cState.JustifiedStreak++
	} else {
		cState.JustifiedStreak = 0
	}

	// update lastFinalizedSlot
	cycleLength := params.GetConfig().CycleLength
	// note: equivalent to justifiedStreak >= cycleLength + 1
	if cState.JustifiedStreak > cycleLength {
		newFinalizedSlot := slot - cycleLength - 1
		if newFinalizedSlot > cState.LastFinalizedSlot {
			cState.LastFinalizedSlot = newFinalizedSlot
		}
	}
}

// quadraticPenaltyQuotient is the quotient that will be used to apply penalties to offline
// validators.
func quadraticPenaltyQuotient() uint64 {
	dropTimeFactor := params.GetConfig().SqrtExpDropTime / params.GetConfig().SlotDuration
	return dropTimeFactor * dropTimeFactor
}

// ProcessSpecialRecords processes the pending special record objects,
// this is called during crystallized state transition.
func processSpecialRecords(actives []*pb.ValidatorRecord, pendingSpecials []*pb.SpecialRecord, slotNumber uint64) {
	// For each special record object in active state.
	for _, specialRecord := range pendingSpecials {
		// Covers validators submitted logouts from last cycle.
		if specialRecord.Kind == uint32(params.Logout) {
			index := binary.BigEndian.Uint64(specialRecord.Data[0])
			// TODO(#633): Verify specialRecord.Data[1] as signature. BLSVerify(pubkey=validator.pubkey, msg=hash(LOGOUT_MESSAGE + bytes8(version))
			exitValidator(actives[index], slotNumber, false)
		}

		// Covers RANDAO updates for all the validators from last cycle.
		if specialRecord.Kind == uint32(params.RandaoChange) {
			index := binary.BigEndian.Uint64(specialRecord.Data[0])
			actives[index].RandaoCommitment = specialRecord.Data[1]
		}
	}
}

// exitValidatorsBelowMinBalance checks if a validator deposit has fallen below min online deposit size,
// it exits the validator if it's below.
func exitValidatorsBelowMinBalance(actives []*pb.ValidatorRecord, slot uint64) {
	minDepositInGWei := params.GetConfig().MinDeposit * params.GetConfig().Gwei

	for index, validator := range actives {
		if validator.Balance < minDepositInGWei {
			actives[index] = exitValidator(validator, slot, false)
		}
	}
}

// exitValidator exits validator from the active list. It returns
// updated validator record with an appropriate status of each validator.
func exitValidator(
	validator *pb.ValidatorRecord,
	currentSlot uint64,
	panalize bool) *pb.ValidatorRecord {
	// TODO(#614): Add validator set change
	validator.ExitSlot = currentSlot
	if panalize {
		validator.Status = uint64(params.Penalized)
	} else {
		validator.Status = uint64(params.PendingExit)
	}
	return validator
}
