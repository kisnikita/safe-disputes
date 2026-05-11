import { Blockchain, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { toNano } from '@ton/core';
import { Investigation } from '../wrappers/Investigation';
import '@ton/test-utils';

describe('Investigation', () => {
    let blockchain: Blockchain;
    let bet: SandboxContract<TreasuryContract>;
    let p1: SandboxContract<TreasuryContract>;
    let p2: SandboxContract<TreasuryContract>;
    let juror1: SandboxContract<TreasuryContract>;
    let juror2: SandboxContract<TreasuryContract>;
    let juror3: SandboxContract<TreasuryContract>;
    let inv: SandboxContract<Investigation>;
    let evidenceDeadline: bigint;

    beforeEach(async () => {
        blockchain = await Blockchain.create();
        bet = await blockchain.treasury('bet');
        p1 = await blockchain.treasury('p1');
        p2 = await blockchain.treasury('p2');
        juror1 = await blockchain.treasury('juror1');
        juror2 = await blockchain.treasury('juror2');
        juror3 = await blockchain.treasury('juror3');
        evidenceDeadline = BigInt(Math.floor(Date.now() / 1000) + 3600);

        inv = blockchain.openContract(await Investigation.fromInit(
            bet.address,
            p1.address,
            p2.address,
            toNano('2') / toNano('1'),
            evidenceDeadline
        ));
    });

    it('accepts evidence from both sides and opens voting before deadline', async () => {
        await inv.send(
            p1.getSender(),
            { value: toNano('0.05') },
            { $$type: 'ProvideEvidence', hash: 11n }
        );
        await inv.send(
            p2.getSender(),
            { value: toNano('0.05') },
            { $$type: 'ProvideEvidence', hash: 22n }
        );

        const ready = await inv.getEvidenceSubmitted();
        expect(ready).toBe(true);
    });

    it('resolves after reaching N votes and sends callback to bet', async () => {
        // N = floor(1 / 0.3) = 3
        await inv.send(
            p1.getSender(),
            { value: toNano('0.05') },
            { $$type: 'ProvideEvidence', hash: 11n }
        );
        await inv.send(
            p2.getSender(),
            { value: toNano('0.05') },
            { $$type: 'ProvideEvidence', hash: 22n }
        );

        await inv.send(juror1.getSender(), { value: toNano('0.05') }, { $$type: 'JurorVote', option: 1n });
        await inv.send(juror2.getSender(), { value: toNano('0.05') }, { $$type: 'JurorVote', option: 1n });
        const third = await inv.send(juror3.getSender(), { value: toNano('0.05') }, { $$type: 'JurorVote', option: 3n });

        expect(third.transactions).toHaveTransaction({
            from: inv.address,
        });
        expect(third.transactions).toHaveTransaction({
            inMessageBounced: true,
        });
    });
});
