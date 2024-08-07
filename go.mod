module code.dogecoin.org/identity

require code.dogecoin.org/governor v1.0.0
require code.dogecoin.org/gossip v1.0.0

// until radicle supports canonical tags
replace code.dogecoin.org/governor => ../governor
replace code.dogecoin.org/gossip => ../gossip

go 1.18
