import { Blockchain, SandboxContract, TreasuryContract } from '@ton/sandbox';
import { toNano } from '@ton/core';
import { BetMaster } from '../wrappers/BetMaster';
import '@ton/test-utils';

describe('BetMaster', () => {
    const minDeposit = toNano('2');
    const minReward = toNano('1');
    const minStake = toNano('2');

    let blockchain: Blockchain;
    let owner: SandboxContract<TreasuryContract>;
    let creator: SandboxContract<TreasuryContract>;

    beforeEach(async () => {
        blockchain = await Blockchain.create();
        owner = await blockchain.treasury('owner');
        creator = await blockchain.treasury('creator');
    });

    const openMaster = async () => blockchain.openContract(await BetMaster.fromInit(minDeposit, minReward, minStake));
    it('updates params only by owner', async () => {
        const master = await openMaster();
        const ok = await master.send(
            owner.getSender(),
            { value: toNano('0.02') },
            { $$type: 'UpdateParams', minDeposit: toNano('3'), minReward: toNano('1.2'), minStake: toNano('2.5') }
        );
        expect(ok.transactions).toHaveTransaction({
            from: owner.address,
            to: master.address,
            success: true,
        });

        const fail = await master.send(
            creator.getSender(),
            { value: toNano('0.02') },
            { $$type: 'UpdateParams', minDeposit: toNano('4'), minReward: toNano('1.3'), minStake: toNano('2.7') }
        );
        expect(fail.transactions).toHaveTransaction({
            from: creator.address,
            to: master.address,
            success: false,
        });
    });
});
