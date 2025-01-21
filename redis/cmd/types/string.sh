# 1. STRING 값 설정
SET key "Hello, Redis!"

# 2. STRING 값 가져오기
GET key

# 3. STRING 값 업데이트
SET key "Updated String"

# 4. STRING 값 증가 (숫자 값일 경우)
SET my_number 10
INCR my_number

# 5. STRING 값 감소
DECR my_number

# 6. STRING 값에 숫자 더하기
INCRBY my_number 5

# 7. STRING 값에 숫자 빼기
DECRBY my_number 3

# 8. 문자열 길이 확인
STRLEN key
