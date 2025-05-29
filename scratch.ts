// scratch.ts
import { Address } from "@ton/core";
try {
    const a = Address.parse('0QAC4r-gk8nlu2H9Ml0-p4Ub-vKaNFv6OUeWrMd00frQOKnY');
    console.log("parsed:", a.toString());
} catch (e) {
    console.error("parse error:", e);
}
