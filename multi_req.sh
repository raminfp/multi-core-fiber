# Multiple concurrent requests
for i in {1..100}; do
  curl http://localhost:3001/store-request &
done

# Check distribution
#curl http://localhost:3001/distribution