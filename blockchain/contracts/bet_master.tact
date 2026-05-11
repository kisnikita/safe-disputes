import "@stdlib/deploy";
import "./bet.tact";

message(0x1f6f3de1) CreateBet {
    id: Int as int128;
    resultDeadline: Int as uint32;
}

message(0x0796e1fb) UpdateParams {
    minDeposit: Int;
    minReward: Int;
    minStake: Int;
}

contract BetMaster {
    const MinTonForStorage: Int = ton("0.1");
    const TopUpPart: Int = ton("0.01");

    owner: Address;
    minDeposit: Int;
    minReward: Int;
    minStake: Int;

    init(minDeposit: Int, minReward: Int, minStake: Int) {
        require(minDeposit > 0, "minDeposit must be > 0");
        require(minReward > 0, "minReward must be > 0");
        require(minStake > 0, "minStake must be > 0");
        require(minReward <= minDeposit, "minReward cannot be greater than minDeposit");
        require(minStake > self.TopUpPart, "minStake must be greater than TopUpPart");
        self.owner = sender();
        self.minDeposit = minDeposit;
        self.minReward = minReward;
        self.minStake = minStake;
    }

    receive() {}

    receive(msg: UpdateParams) {
        require(sender() == self.owner, "only owner");
        if (msg.minDeposit > 0) { self.minDeposit = msg.minDeposit }
        if msg.minReward > 0 { self.minReward = msg.minReward }
        if msg.minStake > 0 { self.minStake = msg.minStake }
        require(self.minReward <= self.minDeposit, "minReward cannot be greater than minDeposit");
        require(self.minStake > self.TopUpPart, "minStake must be greater than TopUpPart");
    }

    receive(msg: CreateBet) {
        let depositPerSide = max(context().value / 10, self.minDeposit);
        let stakePerSide = context().value - depositPerSide;
        let potentialJurorsCount = max(depositPerSide / self.minReward, 1);
        require(stakePerSide >= self.minStake, "stake is below the minimum required");

        // keep peace of stake for contract storage if storage demand.
        if (myBalance() - context().value < self.MinTonForStorage) {
            stakePerSide -= self.TopUpPart;
        }

        deploy(DeployParameters {
            init: initOf Bet(msg.id, sender(), msg.resultDeadline),
            value: stakePerSide + depositPerSide,
            bounce: true,
            mode: SendPayFwdFeesSeparately,
            body: CreateStake {
                stakePerSide,
                depositPerSide,
                potentialJurorsCount,
            }.toCell(),
        });
    }

    get fun minDeposit(): Int {
        return self.minDeposit;
    }

    get fun minReward(): Int {
        return self.minReward;
    }

    get fun minStake(): Int {
        return self.minStake;
    }
}
