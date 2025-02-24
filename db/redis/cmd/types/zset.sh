# 1. ZSET 생성
ZADD active_users2 1 "value1" 2 "value2" 3 "value3"

# 2. ZSET 조회 (전체 조회)
ZRANGE active_users2 0 -1 WITHSCORES

# 3. ZSET 조회 (특정 점수 범위)
ZRANGEBYSCORE active_users2 1 2 WITHSCORES

# 4. ZSET에 새로운 값 추가
ZADD active_users2 4 "value4"

# 5. ZSET 특정 멤버의 점수 업데이트
ZADD active_users2 5 "value1"

# 6. ZSET 특정 멤버 제거
ZREM active_users2 "value2"

# 7. ZSET 크기 확인
ZCARD active_users2

# 8. ZSET 멤버의 점수 확인
ZSCORE active_users2 "value1"

# 9. ZSET 멤버 삭제 (점수 범위로 삭제)
ZREMRANGEBYSCORE active_users2 4 5

# 10. ZSET 조회 (역순 조회)
ZREVRANGE active_users2 0 -1 WITHSCORES
