import { toNano } from '@ton/core';
import { BetMaster } from '../wrappers/BetMaster';
import { NetworkProvider } from '@ton/blueprint';

export async function run(provider: NetworkProvider) {
    const betMaster = provider.open(await BetMaster.fromInit(1514214125n));

    await betMaster.send(
        provider.sender(),
        {
            value: toNano('0.05'),
        },
        null,
    );

    await provider.waitForDeploy(betMaster.address);

    // run methods on `betMaster`
}
