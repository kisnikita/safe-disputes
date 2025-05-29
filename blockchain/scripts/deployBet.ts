import { toNano } from '@ton/core';
import { Bet } from '../wrappers/Bet';
import { NetworkProvider } from '@ton/blueprint';

export async function run(provider: NetworkProvider) {
    const bet = provider.open(await Bet.fromInit(125412515n));

    await bet.send(
        provider.sender(),
        {
            value: toNano('0.05'),
        },
        {
            $$type: 'Deploy',
            queryId: 0n,
        }
    );

    await provider.waitForDeploy(bet.address);

    // run methods on `bet`
}
