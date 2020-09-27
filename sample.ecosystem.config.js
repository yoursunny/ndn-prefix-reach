const IP = "::1";

module.exports = {
  apps: [
    {
      name: "ndn-prefix-reach",
      script: "./ndn-prefix-reach",
      args: `--listen '[${IP}]:8443' --https`,
    },
  ],
};
