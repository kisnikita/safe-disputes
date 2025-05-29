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
        const contract = Bet.fromAddress(Address.parse("kQDGV8b1FlBVodFpTenYb0lvxQgF9CzM0l48OmZf7lytiVpp"));

        return client.open(contract) as OpenedContract<Bet>;
    }, [client])
    
    return {
        deposit: (value: string) => {
            betContract?.send(sender, {
                value: toNano(value)
            }, 
            "deposit")
        },
        accept: (value: string) => {
            betContract?.send(sender, {
                value: toNano(value)
            }, 
            "accept")
        },
        refund: () => {
            betContract?.send(sender, {
                value: toNano('0.02')
            }, 
            "refund")
        },
        win: (adress: Address) => {
            const msg: Winner = {
                $$type: 'Winner',
                address: adress,
            }
            betContract?.send(sender, {
                value: toNano('0.02')
            }, msg)
        }

    }
}