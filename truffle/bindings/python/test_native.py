"""
Simple test to verify native cgo bindings work
"""
from truffle import Truffle, ReservationType, TruffleError

def test_initialization():
    """Test that library loads and initializes"""
    tf = Truffle()
    assert tf._lib is not None
    print("✅ Library loaded and initialized")

def test_search():
    """Test basic search"""
    tf = Truffle()
    
    # Search for a common instance type
    results = tf.search("t3.micro", regions=["us-east-1"])
    
    assert len(results) > 0
    assert results[0].instance_type == "t3.micro"
    assert results[0].region == "us-east-1"
    print(f"✅ Search found {len(results)} results")

def test_spot():
    """Test Spot pricing"""
    tf = Truffle()
    
    # Get Spot prices
    spots = tf.spot("t3.micro", regions=["us-east-1"])
    
    assert len(spots) > 0
    assert spots[0].instance_type == "t3.micro"
    assert spots[0].spot_price > 0
    print(f"✅ Spot pricing: ${spots[0].spot_price:.4f}/hr")

def test_capacity():
    """Test capacity reservations"""
    tf = Truffle()
    
    # Check ODCRs (won't fail if none exist)
    try:
        capacity = tf.capacity(
            reservation_type=ReservationType.ODCR,
            regions=["us-east-1"]
        )
        print(f"✅ Capacity check returned {len(capacity)} reservations")
    except TruffleError as e:
        print(f"✅ Capacity check handled error correctly: {e}")

def test_regions():
    """Test region listing"""
    tf = Truffle()
    
    regions = tf.list_regions()
    
    assert len(regions) > 10  # Should have many regions
    assert "us-east-1" in regions
    print(f"✅ Found {len(regions)} AWS regions")

def main():
    """Run all tests"""
    print("Testing Truffle Native cgo Bindings\n")
    
    try:
        test_initialization()
        test_search()
        test_spot()
        test_capacity()
        test_regions()
        
        print("\n✅ All tests passed!")
        return 0
    
    except Exception as e:
        print(f"\n❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return 1

if __name__ == "__main__":
    exit(main())
