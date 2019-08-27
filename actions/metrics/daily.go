package main

const appendDailyQuery = ` 
SELECT
	d1.job_date_time,
	d1.date,
	d1.address,
	d1.username,
	d1.community,
	d1.community_name,
	d1.balance,
	d1.stake_earned  - IFNULL(d0.stake_earned,0) stake_earned,
	d1.claims_created - IFNULL(d0.claims_created,0) claims_created,
	d1.claims_opened - IFNULL(d0.claims_opened,0) claims_opened,
	d1.unique_claims_opened - IFNULL(d0.unique_claims_opened,0) unique_claims_opened,
	d1.arguments_created - IFNULL(d0.arguments_created,0) arguments_created,
	d1.agrees_received - IFNULL(d0.agrees_received,0) agrees_received,
	d1.agrees_given - IFNULL(d0.agrees_given,0) agrees_given,
	d1.staked - IFNULL(d0.staked,0) staked,
	d1.staked_arguments - IFNULL(d0.staked_arguments,0) staked_arguments,
	d1.staked_agrees - IFNULL(d0.staked_agrees,0) staked_agrees,
	d1.interest_argument_creation - IFNULL(d0.interest_argument_creation,0) interest_argument_creation,
	d1.interest_agree_received - IFNULL(d0.interest_agree_received,0) interest_agree_received,
	d1.interest_agree_given - IFNULL(d0.interest_agree_given,0) interest_agree_given,
	d1.reward_not_helpful - IFNULL(d0.reward_not_helpful,0) reward_not_helpful,
	d1.interest_slashed - IFNULL(d0.interest_slashed,0) interest_slashed,
	d1.stake_slashed - IFNULL(d0.stake_slashed,0) stake_slashed,
	d1.pending_stake - IFNULL(d0.stake_slashed,0) pending_stake
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
    ON d1.address=d0.address and d1.community=d0.community  
`

const appendClaimMetricsDailyQuery = `
SELECT 
d1.job_date_time, 
d1.date,
d1.created_date,
d1.flagged,
d1.id,
d1.community_id,
d1.claim_name,
d1.arguments_created - IFNULL(d0.arguments_created,0) arguments_created,
d1.agrees_given - IFNULL(d0.agrees_given,0) agrees_given,
d1.staked - IFNULL(d0.staked,0) staked,
d1.staked_backed - IFNULL(d0.staked_backed,0) staked_backed,
d1.staked_argument_backed - IFNULL(d0.staked_argument_backed, 0) staked_argument_backed,
d1.staked_agree_backed - IFNULL(d0.staked_agree_backed, 0) staked_agree_backed,
d1.staked_challenged - IFNULL(d0.staked_challenged,0) staked_challenged,
d1.staked_argument_challenged - IFNULL(d0.staked_argument_challenged, 0) staked_argument_challenged,
d1.staked_agree_challenged - IFNULL(d0.staked_agree_challenged, 0) staked_agree_challenged,
d1.user_views - IFNULL(d0.user_views, 0) user_views,
d1.unique_user_views - IFNULL(d0.unique_user_views, 0) unique_user_views,
d1.anon_views - IFNULL(d0.anon_views, 0) anon_views,
d1.unique_anon_views - IFNULL(d0.unique_anon_views, 0) unique_anon_views,
d1.user_arguments_views - IFNULL(d0.user_arguments_views, 0) user_arguments_views,
d1.unique_user_arguments_views - IFNULL(d0.unique_user_arguments_views, 0) unique_user_arguments_views,
d1.anon_arguments_views - IFNULL(d0.anon_arguments_views, 0) anon_arguments_views,
d1.unique_anon_arguments_views - IFNULL(d0.unique_anon_arguments_views, 0) unique_anon_arguments_views,
d1.replies - IFNULL(d0.replies, 0) replies,
d1.last_activiy_argument last_activity_argument,
d1.last_activity_agree last_activity_agree
FROM 

(select * from ` + "`:source_table:`" + ` as x1 where x1.date = ':end_date:') as d1
left join 
(select * from ` + "`:source_table:`" + ` as x2 where x2.date = ':start_date:') as d0
  ON d1.id = d0.id
`
