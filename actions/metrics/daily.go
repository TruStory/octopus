package main

const appendDailyQuery = ` 
SELECT
  d1.date,
  d1.address,
  d1.username,
  d1.category,
  d1.balance,
  d1.cred_earned - IFNULL(d0.cred_earned,0) cred_earned,
  d1.claims_created - IFNULL(d0.claims_created,0) claims_created,
  d1.claims_opened - IFNULL(d0.claims_opened,0) claims_opened,
  d1.unique_claims_opened - IFNULL(d0.unique_claims_opened,0) unique_claims_opened,
  d1.arguments_created - IFNULL(d0.arguments_created,0) arguments_created,
  d1.endorsements_given - IFNULL(d0.endorsements_given,0) endorsements_given,
  d1.endorsements_received - IFNULL(d0.endorsements_received,0) endorsements_received,
  d1.amount_earned - IFNULL(d0.amount_earned,0) amount_earned,
  d1.interest_earned - IFNULL(d0.interest_earned,0) interest_earned,
  d1.amount_lost - IFNULL(d0.amount_lost,0) amount_lost,
  d1.amount_staked - IFNULL(d0.amount_staked,0) amount_staked,
  d1.amount_at_stake
FROM (
  SELECT
    *
  FROM
    ` + "`:source_table:`" + ` as x1
  WHERE
    x1.date=':end_date:' ) AS d1
    
   LEFT JOIN
   (
  SELECT
    *
  FROM
   ` + "`:source_table:`" + ` as x0
  WHERE
    x0.date=':start_date:') as d0
    ON d1.address=d0.address and d1.category=d0.category  
`
