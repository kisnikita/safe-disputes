import { Blockchain, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { toNano } from '@ton/core';
import { Bet } from '../wrappers/Bet';
import '@ton/test-utils';

describe('Bet', () => {
    let blockchain: Blockchain;
    let p1: SandboxContract<TreasuryContract>;
    let p2: SandboxContract<TreasuryContract>;
    let bet: SandboxContract<Bet>;
    let deadline: bigint;

    beforeEach(async () => {
        blockchain = await Blockchain.create();
        p1 = await blockchain.treasury('p1');
        p2 = await blockchain.treasury('p2');
        deadline = BigInt(Math.floor(Date.now() / 1000) + 3600);
        bet = blockchain.openContract(await Bet.fromInit(
            1n,
            p1.address,
            deadline,
        ));
    });

    const createStake = async () =>
        bet.send(
            p1.getSender(),
            { value: toNano('2') },
            {
                $$type: 'CreateStake',
                stakePerSide: toNano('1.8'),
                depositPerSide: toNano('0.2'),
                potentialJurorsCount: 4n,
            }
        );

    const accept = async () =>
        bet.send(
            p2.getSender(),
            { value: toNano('2') },
            { $$type: 'Accept' }
        );

    it('allows cancel before accept', async () => {
        await createStake();
        const cancelRes = await bet.send(
            p1.getSender(),
            { value: toNano('0.02') },
            { $$type: 'Cancel' }
        );
        expect(cancelRes.transactions).toHaveTransaction({
            from: p1.address,
            to: bet.address,
            success: true,
        });
        expect(cancelRes.transactions).toHaveTransaction({
            from: bet.address,
            success: true,
        });
        expect(await bet.getP1Claimable()).toBe(0n);
    });

    it('accepts and resolves direct win', async () => {
        await createStake();
        await accept();

        await bet.send(
            p1.getSender(),
            { value: toNano('0.02') },
            { $$type: 'VoteResult', result: 1n }
        );
        await bet.send(
            p2.getSender(),
            { value: toNano('0.02') },
            { $$type: 'VoteResult', result: 0n }
        );

        const status = await bet.getStatus();
        expect(status).toBe(4n);
        expect(await bet.getP1Claimable()).toBeGreaterThan(0n);
        expect(await bet.getP2Claimable()).toBeGreaterThan(0n);

        const claimP1 = await bet.send(p1.getSender(), { value: toNano('0.02') }, { $$type: 'Claim' });
        expect(claimP1.transactions).toHaveTransaction({
            from: bet.address,
            to: p1.address,
            success: true,
        });
        const claimP2 = await bet.send(p2.getSender(), { value: toNano('0.02') }, { $$type: 'Claim' });
        expect(claimP2.transactions).toHaveTransaction({
            from: bet.address,
            to: p2.address,
            success: true,
        });
    });

    it('accepts and resolves draw (lose + lose)', async () => {
        await createStake();
        await accept();

        await bet.send(
            p1.getSender(),
            { value: toNano('0.02') },
            { $$type: 'VoteResult', result: 0n }
        );
        await bet.send(
            p2.getSender(),
            { value: toNano('0.02') },
            { $$type: 'VoteResult', result: 0n }
        );

        const status = await bet.getStatus();
        expect(status).toBe(4n);
    });
});
