script {
    use std::BasicCoin;

    fun main<CoinType, T>(me: signer) {

        BasicCoin::mint<CoinType>(me, 200);
    }
}
