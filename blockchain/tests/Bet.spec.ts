import { Blockchain, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { toNano } from '@ton/core';
import { Bet, Winner } from '../wrappers/Bet';
import '@ton/test-utils';

describe('Bet', () => {
    let blockchain: Blockchain;
    let firstPlayer: SandboxContract<TreasuryContract>;
    let secondPlayer: SandboxContract<TreasuryContract>;
    let bet: SandboxContract<Bet>;

    beforeEach(async () => { 
        blockchain = await Blockchain.create();

        bet = blockchain.openContract(await Bet.fromInit(1000n));
        
        firstPlayer = await blockchain.treasury('firstPlayer');
        secondPlayer = await blockchain.treasury('secondPlayer');

        const deployResult = await bet.send(
            firstPlayer.getSender(),
            {
                value: toNano('0.05'),
            },
            {
                $$type: 'Deploy',
                queryId: 0n,
            }
        );

        expect(deployResult.transactions).toHaveTransaction({
            from: firstPlayer.address,
            to: bet.address,
            deploy: true,
            success: true,
        });

        await bet.send(
            firstPlayer.getSender(),
            {
                value: toNano('500'),
            },
            "deposit"
        )
    });

    it('should deploy', async () => {
        // the check is done inside beforeEach
        // blockchain and bet are ready to use
    });

    it('should deposit', async () => {
        const statusAfter = await bet.getStatus()
        expect(statusAfter).toBe(1n);
    });

    it('should accept', async () => {
        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('500'),
            },
            "accept"
        )

        const status = await bet.getStatus()
        expect(status).toBe(2n);
    });

    it('should refund firstPlayer', async () => {
        const balanceBefore = await firstPlayer.getBalance()
        await bet.send(
            firstPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            "refund"
        )

        const balanceAfter = await firstPlayer.getBalance()
        expect(balanceAfter).toBeGreaterThan(balanceBefore);
    });

    it('should refund secondPlayer', async () => {
        const balanceBefore = await firstPlayer.getBalance()
        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            "refund"
        )

        const balanceAfter = await firstPlayer.getBalance()
        expect(balanceAfter).toBeGreaterThan(balanceBefore);
    });

    it('should win firstPlayer', async () => {
        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('500'),
            },
            "accept"
        )

        const balanceBefore = await firstPlayer.getBalance()
        const message: Winner = {
            $$type: 'Winner',
            address: firstPlayer.address,
        }
        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            message
        )

        const balanceAfter = await firstPlayer.getBalance()
        expect(balanceAfter).toBeGreaterThan(balanceBefore);
    });

    it('should win secondPlayer', async () => {
        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('500'),
            },
            "accept"
        )
        
        const balanceBefore = await secondPlayer.getBalance()
        const message: Winner = {
            $$type: 'Winner',
            address: secondPlayer.address,
        }
        await bet.send(
            firstPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            message
        )

        const balanceAfter = await secondPlayer.getBalance()
        expect(balanceAfter).toBeGreaterThan(balanceBefore);
    });
});
