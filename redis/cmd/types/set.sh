# 1. SET 생성 및 값 추가
SADD active_users "value1" "value2" "value3"

# 2. SET의 모든 값 가져오기
SMEMBERS active_users

# 3. SET의 특정 값 제거
SREM active_users "value2"

# 4. SET에서 특정 값 확인
SISMEMBER active_users "value1" # 결과: 1(존재), 0(없음)

# 5. SET의 크기 확인
SCARD active_users

# 6. 두 SET의 교집합
SADD active_users2 "value3" "value4"
SINTER active_users active_users2

# 7. 두 SET의 합집합
SUNION active_users active_users2

# 8. 두 SET의 차집합
SDIFF active_users active_users2

# 9. SET에서 랜덤 값 가져오기
SRANDMEMBER active_users

# 10. SET에서 랜덤 값 가져오면서 삭제
SPOP active_users
