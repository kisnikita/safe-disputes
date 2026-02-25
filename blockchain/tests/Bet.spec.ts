import { Blockchain, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { toNano } from '@ton/core';
import { Bet, Deposit, Winner } from '../wrappers/Bet';
import '@ton/test-utils';

describe('Bet', () => {
    let blockchain: Blockchain;
    let firstPlayer: SandboxContract<TreasuryContract>;
    let secondPlayer: SandboxContract<TreasuryContract>;
    let master: SandboxContract<TreasuryContract>;
    let bet: SandboxContract<Bet>;

    beforeEach(async () => {
        blockchain = await Blockchain.create();

        bet = blockchain.openContract(await Bet.fromInit(1000n));

        firstPlayer = await blockchain.treasury('firstPlayer');
        secondPlayer = await blockchain.treasury('secondPlayer');
        master = await blockchain.treasury('master');
    });

    const depositFromMaster = async (value: string) => {
        const msg: Deposit = {
            $$type: 'Deposit',
            player: firstPlayer.address,
        };
        return bet.send(
            master.getSender(),
            {
                value: toNano(value),
            },
            msg
        );
    };

    it('should deposit via message', async () => {
        await depositFromMaster('1');

        const statusAfter = await bet.getStatus();
        const amountAfter = await bet.getAmount();
        expect(statusAfter).toBe(1n);
        expect(amountAfter).toBe(toNano('1'));
    });

    it('should accept', async () => {
        await depositFromMaster('1');

        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('1'),
            },
            'accept'
        );

        const status = await bet.getStatus();
        expect(status).toBe(2n);
    });

    it('should reject accept from the same player', async () => {
        await depositFromMaster('1');

        const acceptResult = await bet.send(
            firstPlayer.getSender(),
            {
                value: toNano('1'),
            },
            'accept'
        );

        expect(acceptResult.transactions).toHaveTransaction({
            from: firstPlayer.address,
            to: bet.address,
            success: false,
        });
    });

    it('should accept with any amount and increase total pot', async () => {
        await depositFromMaster('1');

        const acceptResult = await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('2'),
            },
            'accept'
        );

        expect(acceptResult.transactions).toHaveTransaction({
            from: secondPlayer.address,
            to: bet.address,
            success: true,
        });
        expect(await bet.getStatus()).toBe(2n);
        expect(await bet.getAmount()).toBe(toNano('3'));
    });

    it('should refund after deposit', async () => {
        await depositFromMaster('1');

        const balanceBefore = await firstPlayer.getBalance();
        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            'refund'
        );

        const balanceAfter = await firstPlayer.getBalance();
        expect(balanceAfter).toBeGreaterThan(balanceBefore);
    });

    it('should complete win payout', async () => {
        await depositFromMaster('1');

        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('1'),
            },
            'accept'
        );

        const balanceBefore = await firstPlayer.getBalance();
        const message: Winner = {
            $$type: 'Winner',
            address: firstPlayer.address,
        };
        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            message
        );

        const balanceAfter = await firstPlayer.getBalance();
        expect(balanceAfter).toBeGreaterThan(balanceBefore);
        await expect(bet.getStatus()).rejects.toThrow('non-active');
    });

    it('should split on draw', async () => {
        await depositFromMaster('1');

        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('1'),
            },
            'accept'
        );

        await bet.send(
            firstPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            'draw'
        );

        await bet.send(
            secondPlayer.getSender(),
            {
                value: toNano('0.02'),
            },
            'draw'
        );

        await expect(bet.getStatus()).rejects.toThrow('non-active');
    });
});
