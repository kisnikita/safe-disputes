import { Bet, Winner } from "../../../../blockchain/wrappers/Bet";
import { Address, OpenedContract, toNano } from "@ton/core";
import { useTonClient } from "./useTonClient";
import { useTonConnect } from "./useTonConnect";


export function useBetContract() {
    const { client } = useTonClient();
    const { sender } = useTonConnect()

    const openBet = async (contractAddress: string): Promise<OpenedContract<Bet>> => {
        if (!client) throw new Error("ton client not ready");
        const contract = Bet.fromAddress(Address.parse(contractAddress));
        return client.open(contract) as OpenedContract<Bet>;
    };

    return {
        getAddress: async (betID: bigint) => {
            return (await Bet.fromInit(betID)).address;
        },
        accept: async (contractAddress: string, value: string) => {
            const betContract = await openBet(contractAddress);
            await betContract.send(sender, {
                value: toNano(value)
            },
                "accept");
        },
        refund: async (contractAddress: string) => {
            const betContract = await openBet(contractAddress);
            await betContract.send(sender, {
                value: toNano('0.01')
            },
                "refund");
        },
        win: async (contractAddress: string) => {
            const msg: Winner = {
                $$type: 'Winner',
                address: sender.address ? sender.address :
                    (() => { throw new Error("wallet address not found") })(),
            }
            const betContract = await openBet(contractAddress);
            await betContract.send(sender, {
                value: toNano('0.01')
            }, msg);
        },
        draw: async (contractAddress: string) => {
            const betContract = await openBet(contractAddress);
            await betContract.send(sender, {
                value: toNano('0.01')
            },
                "draw");
        }
    }
}
