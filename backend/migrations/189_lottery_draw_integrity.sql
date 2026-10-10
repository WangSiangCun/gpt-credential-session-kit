ALTER TABLE lottery_draws
    ADD CONSTRAINT lottery_draws_amount_nonnegative_check
        CHECK (amount >= 0),
    ADD CONSTRAINT lottery_draws_result_consistency_check
        CHECK (
            (status = 'lost' AND prize_id IS NULL AND amount = 0)
            OR
            (status = 'won' AND prize_id IS NOT NULL AND amount >= 0)
        );
