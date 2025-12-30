import { toNano } from '@ton/core';
import { Bet, Deposit } from '../wrappers/Bet';
import { NetworkProvider } from '@ton/blueprint';

export async function run(provider: NetworkProvider) {
    const bet = provider.open(await Bet.fromInit(125412515n));
    const sender = provider.sender();
    if (!sender.address) {
        throw new Error('Sender address not found');
    }

    await bet.send(
        sender,
        {
            value: toNano('1'),
        }
        ,
        {
            $$type: 'Deposit',
            player: sender.address,
        } satisfies Deposit
    );

    await provider.waitForDeploy(bet.address);

    // run methods on `bet`
}
