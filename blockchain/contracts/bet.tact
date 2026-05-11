import "@stdlib/deploy";
import "./investigation.tact";

message(0x6a8ec55a) CreateStake {
    stakePerSide: Int;
    depositPerSide: Int;
    potentialJurorsCount: Int;
}
message(0x59b045f8) Accept {}
message(0x368dde63) Cancel {}
message(0xd1f45f36) Claim {}

message(0x75caa6a0) VoteResult { result: Int as uint8 } // 0 lose, 1 win 
message(0x9e2071c5) Finalize {}

contract Bet {
    const StatusPending: Int = 1;
    const StatusAccepted: Int = 2;
    const StatusInvestigation: Int = 3;
    const StatusFinished: Int = 4;

    id: Int as int128;
    p1: Address;
    p2: Address?;
    resultDeadline: Int as uint32;
    status: Int as uint8;
    stakePerSide: Int;
    depositPerSide: Int;
    potentialJurorsCount: Int;

    p1Vote: Int as int8; // -1 unset, 0 lose, 1 win
    p2Vote: Int as int8; // -1 unset, 0 lose, 1 win

    investigationAddress: Address?;
    result: Int as int8; // -1 unset, 0 undecided, 1 p1, 2 p2, 3 draw
    p1Claimable: Int;
    p2Claimable: Int;

    init(id: Int as int128, p1: Address, resultDeadline: Int as uint32) {
        self.id = id;
        self.p1 = p1;
        self.p2 = null;
        self.resultDeadline = resultDeadline;
        self.status = self.StatusPending;
        self.stakePerSide = 0;
        self.depositPerSide = 0;
        self.potentialJurorsCount = 0;
        self.p1Vote = -1;
        self.p2Vote = -1;
        self.investigationAddress = null;
        self.result = -1;
        self.p1Claimable = 0;
        self.p2Claimable = 0;
    }

    receive(msg: CreateStake) {
        require(self.status == self.StatusPending, "bad status");
        require(self.stakePerSide == 0, "already funded");
        require(msg.stakePerSide > 0, "bad stake");
        require(msg.depositPerSide > 0, "bad deposit");
        require(msg.potentialJurorsCount > 0, "bad requiredVotesCount");
        require(context().value == msg.stakePerSide + msg.depositPerSide, "bad initial value");

        self.stakePerSide = msg.stakePerSide;
        self.depositPerSide = msg.depositPerSide;
        self.potentialJurorsCount = msg.potentialJurorsCount;
    }

    receive(_: Accept) {
        require(self.status == self.StatusPending, "bad status");
        require(self.stakePerSide > 0, "stake not set");
        require(sender() != self.p1, "p1 cannot accept");
        require(context().value == self.stakePerSide + self.depositPerSide, "stakes are different");
        self.p2 = sender();
        self.status = self.StatusAccepted;
    }

    receive(_: Cancel) {
        require(self.status == self.StatusPending, "cannot cancel");
        require(sender() == self.p1, "only p1");
        self.status = self.StatusFinished;
        message(MessageParameters {
            to: self.p1,
            value: 0,
            bounce: false,
            mode: SendRemainingBalance,
        });
    }

    receive(_: Claim) {
        require(self.status == self.StatusFinished, "bad status");
        if (sender() == self.p1) {
            let value = self.p1Claimable;
            require(value > 0, "nothing to claim");
            self.p1Claimable = 0;
            message(MessageParameters {
                to: self.p1,
                value,
                bounce: false,
            });
            return;
        }

        require(self.p2 != null && sender() == self.p2!!, "only participant");
        let value = self.p2Claimable;
        require(value > 0, "nothing to claim");
        self.p2Claimable = 0;
        message(MessageParameters {
            to: self.p2!!,
            value,
            bounce: false,
        });
    }

    receive(msg: VoteResult) {
        require(self.status == self.StatusAccepted, "bad status");
        require(now() < self.resultDeadline, "deadline passed");
        require(msg.result == 0 || msg.result == 1, "bad result");

        if (sender() == self.p1) {
            require(self.p1Vote == -1, "already voted");
            self.p1Vote = msg.result;
        } else {
            require(self.p2 != null && sender() == self.p2!!, "only participant");
            require(self.p2Vote == -1, "already voted");
            self.p2Vote = msg.result;
        }

        if (self.p1Vote != -1 && self.p2Vote != -1) { self.settleFromVotes() }
    }

    receive(msg: InvestigationResolved) {
        require(self.status == self.StatusInvestigation, "bad status");
        require(self.investigationAddress != null, "no investigation");
        require(sender() == self.investigationAddress!!, "only investigation");
        require(msg.finalDecision >= 0 && msg.finalDecision <= 3, "bad finalDecision");

        self.result = msg.finalDecision;
        self.settleAfterInvestigation();
    }

    inline fun settleAfterInvestigation() {
        require(self.status == self.StatusInvestigation, "bad status");
        if (self.result == 1) {
            self.status = self.StatusFinished;
            self.p1Claimable = distributableBalance();
            return;
        }
        if (self.result == 2) {
            self.status = self.StatusFinished;
            self.p2Claimable = distributableBalance();
            return;
        }
        self.settleDraw();
    }

    receive(_: Finalize) {
        require(self.status == self.StatusAccepted, "bad status");
        require(now() >= self.resultDeadline, "deadline not reached");

        if (self.p1Vote == -1) { self.p1Vote = 0 }
        if (self.p2Vote == -1) { self.p2Vote = 0 }
        self.settleFromVotes();
    }

    fun settleFromVotes() {
        require(self.p1Vote != -1 && self.p2Vote != -1, "votes are not complete");

        if (self.p1Vote == 1 && self.p2Vote == 1) {
            self.result = -1;
            self.createInvestigation();
            return;
        }

        if (self.p1Vote == 1 && self.p2Vote == 0) {
            self.result = 1;
            self.settleWinLose(self.p1, self.p2!!);
            return;
        }

        if (self.p1Vote == 0 && self.p2Vote == 1) {
            self.result = 2;
            self.settleWinLose(self.p2!!, self.p1);
            return;
        }

        self.result = 3;
        self.settleDraw();
    }

    inline fun createInvestigation() {
        self.status = self.StatusInvestigation;
        require(self.potentialJurorsCount > 0, "requiredVotesCount not set");
        let investigationInit = initOf Investigation(
            myAddress(),
            self.p1,
            self.p2!!,
            self.potentialJurorsCount,
            self.resultDeadline,
        );
        self.investigationAddress = contractAddress(investigationInit);

        deploy(DeployParameters {
            init: investigationInit,
            value: self.depositPerSide,
            bounce: true,
            body: CreateInvestigation {}.toCell(),
        });
    }

    fun settleWinLose(winner: Address, loser: Address) {
        self.status = self.StatusFinished;
        let available = distributableBalance();
        let loserValue = min(self.depositPerSide, available);
        let winnerValue = available - loserValue;
        self.assignClaimable(loser, loserValue);
        self.assignClaimable(winner, winnerValue);
    }

    fun settleDraw() {
        self.status = self.StatusFinished;
        let available = distributableBalance();
        let p1Value = min(self.stakePerSide + self.depositPerSide, available);
        let p2Value = available - p1Value;
        self.p1Claimable = p1Value;
        self.p2Claimable = p2Value;
    }

    inline fun assignClaimable(participant: Address, value: Int) {
        if (participant == self.p1) {
            self.p1Claimable = value;
            return;
        }
        self.p2Claimable = value;
    }

    get fun status(): Int {
        return self.status;
    }

    get fun stake(): Int {
        return self.stakePerSide;
    }

    get fun deposit(): Int {
        return self.depositPerSide;
    }

    get fun stakeWithDeposit(): Int {
        return self.stakePerSide + self.depositPerSide;
    }

    get fun resultDeadline(): Int {
        return self.resultDeadline;
    }

    get fun votes(): Int {
        return self.p1Vote * 10 + self.p2Vote;
    }

    get fun result(): Int {
        return self.result;
    }

    get fun investigationAddress(): Address {
        require(self.investigationAddress != null, "not created");
        return self.investigationAddress!!;
    }

    get fun p1Claimable(): Int {
        return self.p1Claimable;
    }

    get fun p2Claimable(): Int {
        return self.p2Claimable;
    }
}

fun distributableBalance(): Int {
    return max(myBalance() - context().value, 0);
}
