script {
    use std::BasicCoin;

    fun main<CoinType, T>(me: signer, amount: u64) {
        BasicCoin::mint<CoinType>(me, amount);
    }
}
