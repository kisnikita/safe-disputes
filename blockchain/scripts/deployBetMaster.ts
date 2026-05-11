import { toNano } from '@ton/core';
import { BetMaster } from '../wrappers/BetMaster';
import { NetworkProvider } from '@ton/blueprint';

export async function run(provider: NetworkProvider) {
    const sender = provider.sender();
    if (!sender.address) {
        throw new Error('Sender address not found');
    }

    const betMaster = provider.open(await BetMaster.fromInit(
        toNano('0.1'),
        toNano('0.1'),
        toNano('0.05'),
    ));

    await betMaster.send(
        provider.sender(),
        {
            value: toNano('0.01'),
        },
        null,
    );

    await provider.waitForDeploy(betMaster.address);
}
