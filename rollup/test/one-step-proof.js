const { ethers, run, getNamedAccounts } = require("hardhat");
const fs = require("fs");
const assert = require("assert");

const PARALLEL = 128;

async function sendTestMessages(deployment) {
  await deployment;
  const { deployer } = await getNamedAccounts();
  inbox = await ethers.getContract("Inbox", deployer);
  seqInbox = await ethers.getContract("SequencerInbox", deployer);
  const msgRoot = "../prover/test-cases/rust/messages/"
  for (let msgNum = 0;  msgNum < 2; msgNum++) {
    let path = msgRoot + "msg" + String(msgNum) +".bin"
    buf = fs.readFileSync(path);
    inbox.sendL2MessageFromOrigin(buf);
    seqInbox.addSequencerL2BatchFromOrigin(msgNum, buf, "0", "0");
  } 
}

describe("OneStepProof", function () {
  const deployment = run("deploy", { "tags": "OneStepProofEntry" });
  const root = "./test/proofs/";
  const dir = fs.readdirSync(root);

  sendTestMessagesPromise = sendTestMessages(deployment);

  for (let file of dir) {
    if (!file.endsWith(".json")) continue;
    it("Should pass " + file + " proofs", async function () {
      await deployment;

      let path = root + file;
      let proofs = JSON.parse(fs.readFileSync(path));

      const osp = await ethers.getContract("OneStepProofEntry");

      await sendTestMessagesPromise;

      const promises = [];
      for (let i = 0; i < proofs.length; i++) {
        process.stdout.write("\rTesting " + file + " proof " + i + "/" + proofs.length + " ");
        const proof = proofs[i];
        const promise = osp.proveOneStep([...Buffer.from(proof.before, "hex")], [...Buffer.from(proof.proof, "hex")])
          .catch(err => {
            console.error("Error executing proof " + i);
            throw err;
          })
          .then(after => assert.equal(after, "0x" + proof.after, "After state doesn't match after proof " + i));
        if (promises.length < PARALLEL) {
          promises.push(promise);
        } else {
          const finished = await Promise.race(promises.map((p, i) => p.then(_ => i)));
          promises[finished] = promise;
        }
      }
      await Promise.all(promises);
      process.stdout.write("\r");
    });
  }
});
