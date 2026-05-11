message(0x3c27af32) CreateInvestigation {}
message(0x2d172ca4) ProvideEvidence { hash: Int as uint256 }
message(0x5f0d6f2a) JurorVote { option: Int as uint8 } // 1 p1, 2 p2, 3 draw
message(0x4f8f6c42) InvestigationResolved { finalDecision: Int as uint8 } // 0 undecided, 1 p1, 2 p2, 3 draw

contract Investigation {
    betAddress: Address;
    p1: Address;
    p2: Address;

    requiredVotesCount: Int;

    evidenceDeadline: Int as uint32;
    p1EvidenceHash: Int as uint256;
    p2EvidenceHash: Int as uint256;
    evidenceSubmitted: Bool;

    p1Votes: Int;
    drawVotes: Int;
    p2Votes: Int;
    totalVotes: Int;
    isFinalized: Bool;
    jurorVotes: map<Address, Int as uint8>;

    init(
        betAddress: Address,
        p1: Address,
        p2: Address,
        requiredVotesCount: Int,
        evidenceDeadline: Int as uint32,
    ) {
        require(requiredVotesCount > 0, "requiredVotesCount must be > 0");
        self.betAddress = betAddress;
        self.p1 = p1;
        self.p2 = p2;
        self.requiredVotesCount = requiredVotesCount;
        self.evidenceDeadline = evidenceDeadline;
        self.p1EvidenceHash = 0;
        self.p2EvidenceHash = 0;
        self.evidenceSubmitted = false;
        self.p1Votes = 0;
        self.drawVotes = 0;
        self.p2Votes = 0;
        self.totalVotes = 0;
        self.isFinalized = false;
        self.jurorVotes = emptyMap();
    }

    receive(msg: CreateInvestigation) {}

    receive(msg: ProvideEvidence) {
        require(!self.isFinalized, "finalized");
        require(now() < self.evidenceDeadline, "evidence deadline passed");
        if (sender() == self.p1) {
            self.p1EvidenceHash = msg.hash;
        } else if (sender() == self.p2) {
            self.p2EvidenceHash = msg.hash;
        } else {
            require(false, "only participants");
        }
        self.evidenceSubmitted = self.p1EvidenceHash != 0 && self.p2EvidenceHash != 0;
    }

    receive(msg: JurorVote) {
        require(!self.isFinalized, "finalized");
        require(msg.option >= 1 && msg.option <= 3, "bad option");
        require(self.evidenceSubmitted || now() >= self.evidenceDeadline, "evidence phase");
        require(self.totalVotes < self.requiredVotesCount, "enough votes");
        require(!self.jurorVotes.exists(sender()), "already voted");
        self.jurorVotes.set(sender(), msg.option);

        if (msg.option == 1) {
            self.p1Votes += 1;
        } else if (msg.option == 2) {
            self.p2Votes += 1;
        } else {
            self.drawVotes += 1;
        }
        self.totalVotes += 1;

        if (self.totalVotes == self.requiredVotesCount) {
            self.resolveAndPay();
        }
    }

    inline fun resolveAndPay() {
        require(!self.isFinalized, "finalized");
        self.isFinalized = true;

        let topVotes = max(self.p1Votes, max(self.drawVotes, self.p2Votes));
        let topOption = topVotes == self.p1Votes ? 1 : topVotes == self.p2Votes ? 2 : 3;
        let secondVotes = self.totalVotes - topVotes - min(self.p1Votes, min(self.drawVotes, self.p2Votes));

        let decisive = topVotes * 100 >= self.requiredVotesCount * 60 && (topVotes - secondVotes) * 100 >= self.requiredVotesCount * 25;
        let finalDecision = 0;
        if (decisive) {
            finalDecision = topOption;
        }

        if (decisive && topVotes > 0) {
            let rewardPerWinner = max(myBalance() - context().value, 0) / topVotes;
            if (rewardPerWinner > 0) {
                foreach (juror, option in self.jurorVotes) {
                    if (option == topOption) {
                        message(MessageParameters {
                            to: juror,
                            value: rewardPerWinner,
                            bounce: false,
                            mode: SendIgnoreErrors,
                        });
                    }
                }
            }
        }

        message(MessageParameters {
            to: self.betAddress,
            value: 0,
            bounce: true,
            mode: SendRemainingBalance,
            body: InvestigationResolved { finalDecision }.toCell(),
        });
    }

    get fun requiredVotesCount(): Int {
        return self.requiredVotesCount;
    }

    get fun evidenceSubmitted(): Bool {
        return self.evidenceSubmitted;
    }

    get fun totalVotes(): Int {
        return self.totalVotes;
    }
}
