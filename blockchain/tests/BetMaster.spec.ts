import { Blockchain, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { toNano } from '@ton/core';
import { BetMaster, CreateBet } from '../wrappers/BetMaster';
import { Bet } from '../wrappers/Bet';
import '@ton/test-utils';

describe('BetMaster', () => {
    let blockchain: Blockchain;
    let deployer: SandboxContract<TreasuryContract>;
    let master: SandboxContract<BetMaster>;

    beforeEach(async () => {
        blockchain = await Blockchain.create();

        master = blockchain.openContract(await BetMaster.fromInit(1000n));
        deployer = await blockchain.treasury('deployer');

        const deployResult = await master.send(
            deployer.getSender(),
            {
                value: toNano('0.05'),
            },
            null,
        );
        expect(deployResult.transactions).toHaveTransaction({
            from: deployer.address,
            to: master.address,
            deploy: true,
            success: true,
        });
    });

    it('should deploy bet and store address', async () => {
        const betId = 123n;
        const msg: CreateBet = {
            $$type: 'CreateBet',
            id: betId,
        };

        const result = await master.send(
            deployer.getSender(),
            {
                value: toNano('0.05'),
            },
            msg
        );

        const bet = blockchain.openContract(await Bet.fromInit(betId));

        expect(result.transactions).toHaveTransaction({
            from: master.address,
            to: bet.address,
            deploy: true,
            success: true,
        });  

        const status = await bet.getStatus();
        const amount = await bet.getAmount();
        expect(status).toBe(1n);
        expect(amount).toBeGreaterThan(0n);
    });

    it('should reject duplicate bet id', async () => {
        const betId = 555n;
        const bet = blockchain.openContract(await Bet.fromInit(betId));

        await master.send(
            deployer.getSender(),
            {
                value: toNano('0.05'),
            },
            {
                $$type: 'CreateBet',
                id: betId,
            }
        );

        const second = await master.send(
            deployer.getSender(),
            {
                value: toNano('0.05'),
            },
            {
                $$type: 'CreateBet',
                id: betId,
            }
        );

        expect(second.transactions).toHaveTransaction({
            from: master.address,
            to: bet.address,
            success: false,
        });
    });

    it('should not deploy bet with insufficient value', async () => {
        const betId = 777n;
        const bet = blockchain.openContract(await Bet.fromInit(betId));

        const res = await master.send(
            deployer.getSender(),
            {
                value: toNano('0.005'),
            },
            {
                $$type: 'CreateBet',
                id: betId,
            }
        );

        expect(res.transactions).toHaveTransaction({
            from: deployer.address,
            to: master.address,
            success: false,
        });
        expect(res.transactions).not.toHaveTransaction({
            to: bet.address,
            deploy: true,
            success: true,
        });
    });

    it('should not deploy unknown betID by default', async () => {
        const bet = blockchain.openContract(await Bet.fromInit(999n));
        const contract = await blockchain.getContract(bet.address);
        expect(contract.accountState?.type).not.toBe('active');
    });
});
