import { useAsyncInitialize } from "./useAsyncInitialize";
import {Bet, Winner} from "../../../../blockchain/wrappers/Bet";
import { Address, OpenedContract, toNano } from "@ton/core";
import { useTonClient } from "./useTonClient";
import { useTonConnect } from "./useTonConnect";


export function useBetContract() {
    const { client } = useTonClient();
    const {sender} = useTonConnect()
    
    const betContract = useAsyncInitialize(async () => {
        if (!client) return;
        const contract = Bet.fromAddress(Address.parse("EQDH-tF74ugbk1zZWk2Xtm6wu4aUL6JzCTRuO2Hul_XIEuNH"));

        return client.open(contract) as OpenedContract<Bet>;
    }, [client])

    return {
        deposit: async (value: string) => {
            if (!betContract) throw new Error("contract not ready");
            await betContract.send(sender, {
                value: toNano(value)
            }, 
            "deposit");
        },
        accept: async (value: string) => {
            if (!betContract) throw new Error("contract not ready");
            await betContract.send(sender, {
                value: toNano(value)
            }, 
            "accept");
        },
        refund: async () => {
            if (!betContract) throw new Error("contract not ready");
            await betContract.send(sender, {
                value: toNano('0.01')
            }, 
            "refund");
        },
        win: async () => {
            const msg: Winner = {
                $$type: 'Winner',
                address: sender.address ? sender.address : 
                (() => { throw new Error("wallet address not found") })(),
            }
            if (!betContract) throw new Error("contract not ready");
            await betContract.send(sender, {
                value: toNano('0.01')
            }, msg);
        },
        draw: async () => {
            if (!betContract) throw new Error("contract not ready");
            await betContract.send(sender, {
                value: toNano('0.01')
            }, 
            "draw");
        }
    }
}
